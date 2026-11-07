package proxy

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stakater/GitWebhookProxy/pkg/parser"
	"github.com/stakater/GitWebhookProxy/pkg/providers"
	"github.com/stakater/GitWebhookProxy/pkg/utils"
)

var (
	// transport is used for all outbound requests to the upstream. Upstream TLS
	// certificate verification is enabled by default (InsecureSkipVerify:false)
	// and can only be disabled via the explicit --insecureSkipVerify flag.
	transport = &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	}
	httpClient = &http.Client{
		Timeout:   time.Second * 30,
		Transport: transport,
	}
)

// SetInsecureSkipVerify controls whether the upstream's TLS certificate is
// verified when forwarding webhooks. It is false by default; enabling it
// disables certificate and hostname verification and should only be used for
// trusted upstreams presenting self-signed certificates.
func SetInsecureSkipVerify(skip bool) {
	transport.TLSClientConfig.InsecureSkipVerify = skip
}

type Proxy struct {
	provider     string
	upstreamURL  string
	allowedPaths []string
	secret       string
	ignoredUsers []string
	allowedUsers []string
}

func (p *Proxy) isPathAllowed(path string) bool {
	// All paths allowed
	if len(p.allowedPaths) == 0 {
		return true
	}

	// A path is allowed when it equals an allowed path or lies below it.
	// Matching stops at segment boundaries: "/path1" allows "/path1/x" but
	// not "/path10".
	incomingPath := strings.TrimSuffix(strings.TrimSpace(path), "/")
	for _, allowedPath := range p.allowedPaths {
		allowedPath = strings.TrimSuffix(strings.TrimSpace(allowedPath), "/")
		if incomingPath == allowedPath || strings.HasPrefix(incomingPath, allowedPath+"/") {
			return true
		}
	}
	return false
}

func (p *Proxy) isIgnoredUser(committer string) bool {
	if len(p.ignoredUsers) > 0 {
		if exists, _ := utils.InArray(p.ignoredUsers, committer); exists {
			return true
		}
	}

	if committer == "" && p.provider == providers.GithubName {
		return true
	}

	return false
}

func (p *Proxy) isAllowedUser(committer string) bool {
	if len(p.allowedUsers) > 0 {
		if exists, _ := utils.InArray(p.allowedUsers, committer); exists {
			return true
		}

		return false
	}

	return true
}

func (p *Proxy) redirect(hook *providers.Hook, redirectURL string) (*http.Response, error) {
	if hook == nil {
		return nil, errors.New("cannot redirect with nil hook")
	}

	// Parse url to check validity
	url, err := url.Parse(redirectURL)
	if err != nil {
		return nil, err
	}

	// Assign default scheme as http if not specified
	if url.Scheme == "" {
		url.Scheme = "http"
	}

	// Create Redirect request
	req, err := http.NewRequest(hook.RequestMethod, url.String(), bytes.NewBuffer(hook.Payload))

	if err != nil {
		return nil, err
	}

	// Set Headers from hook
	for key, value := range hook.Headers {
		req.Header.Add(key, value)
	}

	return httpClient.Do(req)
}

func (p *Proxy) proxyRequest(w http.ResponseWriter, r *http.Request) {
	redirectURL := p.upstreamURL + r.URL.Path

	if r.URL.RawQuery != "" {
		redirectURL += "?" + r.URL.RawQuery
	}

	// Request values are quoted in log lines so control characters are escaped.
	logURL, logUpstream := strconv.Quote(r.URL.String()), strconv.Quote(redirectURL)
	log.Printf("Proxying Request from %s, to upstream %s\n", logURL, logUpstream)

	if !p.isPathAllowed(r.URL.Path) {
		log.Printf("Not allowed to proxy path: %s", strconv.Quote(r.URL.Path))
		http.Error(w, "Not allowed to proxy path: '"+r.URL.Path+"'", http.StatusForbidden)
		return
	}

	provider, err := providers.NewProvider(p.provider, p.secret)
	if err != nil {
		log.Printf("Error creating provider: %s", err)
		http.Error(w, "Error creating Provider", http.StatusInternalServerError)
		return
	}

	hook, err := parser.Parse(r, provider)
	if err != nil {
		log.Printf("Error Parsing Hook: %s", err)
		http.Error(w, "Error parsing Hook: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate before user filtering so unsigned requests learn nothing
	// about the ignored and allowed user lists.
	if len(strings.TrimSpace(p.secret)) > 0 && !provider.Validate(*hook) {
		log.Printf("Error Validating Hook")
		http.Error(w, "Error validating Hook", http.StatusBadRequest)
		return
	}

	committer := provider.GetCommitter(*hook)
	log.Printf("Incoming request from user: %q", committer)
	if p.isIgnoredUser(committer) || (!p.isAllowedUser(committer)) {
		log.Printf("Ignoring request for user: %q", committer)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprintf(w, "Ignoring request for user: %s", committer); err != nil {
			log.Printf("Error writing response: %v", err)
		}
		return
	}

	resp, errs := p.redirect(hook, redirectURL)
	if errs != nil {
		log.Printf("Error Redirecting %s to upstream %s: %s\n", logURL, logUpstream, errs)
		http.Error(w, "Error forwarding request to upstream", http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		log.Printf("Error Redirecting %s to upstream %s, Upstream Redirect Status: %s\n", logURL, logUpstream, resp.Status)
		http.Error(w, "Upstream returned status: "+resp.Status, resp.StatusCode)
		return
	}

	log.Printf("Redirected incoming request %s to %s with Response: '%s'\n",
		logURL, logUpstream, resp.Status)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error Reading upstream %s response body\n", logURL)
		http.Error(w, "Error reading upstream response body", http.StatusBadGateway)
		return
	}

	w.WriteHeader(resp.StatusCode)
	if _, err := w.Write(responseBody); err != nil {
		log.Printf("Error writing response body: %v", err)
	}
}

// Health Check Endpoint
func (p *Proxy) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(200)
	if _, err := fmt.Fprint(w, "I'm Healthy and I know it! ;) "); err != nil {
		log.Printf("Error writing health check response: %v", err)
	}
}

// Run starts Proxy server
func (p *Proxy) Run(listenAddress string) error {
	if len(strings.TrimSpace(listenAddress)) == 0 {
		panic("Cannot create Proxy with empty listenAddress")
	}

	log.Printf("Listening at: %s", listenAddress)
	return p.newServer(listenAddress).ListenAndServe()
}

// Server timeouts. Upstream requests time out after 30s, so the write timeout
// leaves room for the proxied response.
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 60 * time.Second
	idleTimeout       = 120 * time.Second
)

func (p *Proxy) newServer(listenAddress string) *http.Server {
	return &http.Server{
		Addr:              listenAddress,
		Handler:           p.Handler(),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

// Handler returns the HTTP handler serving the health check, the embedded
// assets and the webhook proxy endpoint.
func (p *Proxy) Handler() http.Handler {
	router := chi.NewRouter()
	router.Get("/health", p.health)
	router.Get("/assets/*", p.serveAssets)
	router.Post("/*", p.proxyRequest)
	return router
}

func NewProxy(upstreamURL string, allowedPaths []string,
	provider string, secret string, ignoredUsers []string, allowedUsers []string) (*Proxy, error) {
	// Validate Params
	if len(strings.TrimSpace(upstreamURL)) == 0 {
		return nil, errors.New("cannot create proxy with empty upstreamURL")
	}
	if len(strings.TrimSpace(provider)) == 0 {
		return nil, errors.New("cannot create proxy with empty provider")
	}
	if allowedPaths == nil {
		return nil, errors.New("cannot create proxy with nil allowedPaths")
	}

	return &Proxy{
		provider:     provider,
		upstreamURL:  upstreamURL,
		allowedPaths: allowedPaths,
		secret:       secret,
		ignoredUsers: ignoredUsers,
		allowedUsers: allowedUsers,
	}, nil
}
