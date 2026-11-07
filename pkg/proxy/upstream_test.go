package proxy

import (
	"bytes"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stakater/GitWebhookProxy/pkg/providers"
)

type recordedRequest struct {
	method   string
	path     string
	rawQuery string
	header   http.Header
	body     []byte
}

// testUpstream is a local stand-in for the webhook receiver. It records every
// request and answers 405 on /get and 200 "ok" everywhere else, unless a
// custom handler is set.
type testUpstream struct {
	*httptest.Server
	mu       sync.Mutex
	received []recordedRequest
}

func newUpstream(t *testing.T) *testUpstream {
	t.Helper()
	return newUpstreamWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/get" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		_, _ = io.WriteString(w, "ok")
	})
}

func newUpstreamWithHandler(t *testing.T, handler http.HandlerFunc) *testUpstream {
	t.Helper()
	u := &testUpstream{}
	u.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		u.mu.Lock()
		u.received = append(u.received, recordedRequest{
			method:   r.Method,
			path:     r.URL.Path,
			rawQuery: r.URL.RawQuery,
			header:   r.Header.Clone(),
			body:     body,
		})
		u.mu.Unlock()
		handler(w, r)
	}))
	t.Cleanup(u.Close)
	return u
}

func (u *testUpstream) requests() []recordedRequest {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]recordedRequest(nil), u.received...)
}

// closedServerURL returns the URL of a server that is no longer listening, so
// connections to it are refused without any network access.
func closedServerURL(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	return srv.URL
}

func newGithubRequest(path, event, signature, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set(providers.XGitHubEvent, event)
	req.Header.Set(providers.XGitHubDelivery, "72d3162e-cc78-11e3-81ab-4c9367dc0958")
	req.Header.Set(providers.ContentTypeHeader, providers.DefaultContentTypeHeaderValue)
	if signature != "" {
		req.Header.Set(providers.XHubSignature, signature)
	}
	return req
}

func githubSignature256(secret, body string) string {
	return providers.Signature256Prefix + providers.HashPayloadSHA256(secret, []byte(body))
}

func withHeader(req *http.Request, key, value string) *http.Request {
	req.Header.Set(key, value)
	return req
}

func githubSignature(secret, body string) string {
	return providers.SignaturePrefix + providers.HashPayload(secret, []byte(body))
}

func TestProxyRequest_Github(t *testing.T) {
	t.Parallel()
	const (
		secret = "github-secret"
		push   = `{"ref":"refs/heads/main","sender":{"login":"octocat"}}`
		ping   = `{"zen":"Design for failure.","sender":{"login":"octocat"}}`
	)

	tests := []struct {
		name          string
		secret        string
		allowedPaths  []string
		ignoredUsers  []string
		allowedUsers  []string
		request       *http.Request
		wantStatus    int
		wantBody      string
		wantForwarded bool
	}{
		{
			name:          "valid signature is forwarded",
			secret:        secret,
			request:       newGithubRequest("/github-webhook/", "push", githubSignature(secret, push), push),
			wantStatus:    http.StatusOK,
			wantBody:      "ok",
			wantForwarded: true,
		},
		{
			name:       "invalid signature is rejected",
			secret:     secret,
			request:    newGithubRequest("/github-webhook/", "push", githubSignature("other-secret", push), push),
			wantStatus: http.StatusBadRequest,
			wantBody:   "Error validating Hook\n",
		},
		{
			name:       "signature over a different body is rejected",
			secret:     secret,
			request:    newGithubRequest("/github-webhook/", "push", githubSignature(secret, push+" "), push),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing signature header is rejected",
			secret:     secret,
			request:    newGithubRequest("/github-webhook/", "push", "", push),
			wantStatus: http.StatusBadRequest,
			wantBody:   "Error validating Hook\n",
		},
		{
			name:       "signature with wrong prefix is rejected",
			secret:     secret,
			request:    newGithubRequest("/github-webhook/", "push", "sha2="+providers.HashPayload(secret, []byte(push)), push),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:          "valid sha256 signature is forwarded",
			secret:        secret,
			request:       withHeader(newGithubRequest("/github-webhook/", "push", "", push), providers.XHubSignature256, githubSignature256(secret, push)),
			wantStatus:    http.StatusOK,
			wantForwarded: true,
		},
		{
			name:       "invalid sha256 signature is rejected despite valid sha1",
			secret:     secret,
			request:    withHeader(newGithubRequest("/github-webhook/", "push", githubSignature(secret, push), push), providers.XHubSignature256, githubSignature256("other-secret", push)),
			wantStatus: http.StatusBadRequest,
			wantBody:   "Error validating Hook\n",
		},
		{
			name:          "no secret configured skips validation",
			secret:        "",
			request:       newGithubRequest("/github-webhook/", "push", "", push),
			wantStatus:    http.StatusOK,
			wantForwarded: true,
		},
		{
			name:       "event without committer is ignored",
			secret:     secret,
			request:    newGithubRequest("/github-webhook/", "ping", githubSignature(secret, ping), ping),
			wantStatus: http.StatusOK,
			wantBody:   "Ignoring request for user: ",
		},
		{
			name:         "ignored user is not forwarded",
			secret:       secret,
			ignoredUsers: []string{"bot", "octocat"},
			request:      newGithubRequest("/github-webhook/", "push", githubSignature(secret, push), push),
			wantStatus:   http.StatusOK,
			wantBody:     "Ignoring request for user: octocat",
		},
		{
			// Signature validation runs before user filtering, so an unsigned
			// request cannot probe the ignored and allowed user lists.
			name:         "invalid signature is rejected before user filtering",
			secret:       secret,
			ignoredUsers: []string{"octocat"},
			request:      newGithubRequest("/github-webhook/", "push", githubSignature("other-secret", push), push),
			wantStatus:   http.StatusBadRequest,
			wantBody:     "Error validating Hook\n",
		},
		{
			name:         "user missing from allowed users is not forwarded",
			secret:       secret,
			allowedUsers: []string{"hubot"},
			request:      newGithubRequest("/github-webhook/", "push", githubSignature(secret, push), push),
			wantStatus:   http.StatusOK,
			wantBody:     "Ignoring request for user: octocat",
		},
		{
			name:          "allowed user is forwarded",
			secret:        secret,
			allowedUsers:  []string{"hubot", "octocat"},
			request:       newGithubRequest("/github-webhook/", "push", githubSignature(secret, push), push),
			wantStatus:    http.StatusOK,
			wantForwarded: true,
		},
		{
			name:         "allowed user with invalid signature is rejected",
			secret:       secret,
			allowedUsers: []string{"octocat"},
			request:      newGithubRequest("/github-webhook/", "push", githubSignature("other-secret", push), push),
			wantStatus:   http.StatusBadRequest,
		},
		{
			name:         "path outside allowed paths is forbidden",
			secret:       secret,
			allowedPaths: []string{"/github-webhook"},
			request:      newGithubRequest("/other", "push", githubSignature(secret, push), push),
			wantStatus:   http.StatusForbidden,
			wantBody:     "Not allowed to proxy path: '/other'\n",
		},
		{
			name:          "path below an allowed path is forwarded",
			secret:        secret,
			allowedPaths:  []string{"/project/a", "/github-webhook"},
			request:       newGithubRequest("/github-webhook/job/build", "push", githubSignature(secret, push), push),
			wantStatus:    http.StatusOK,
			wantForwarded: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			upstream := newUpstream(t)
			allowedPaths := tt.allowedPaths
			if allowedPaths == nil {
				allowedPaths = []string{}
			}
			p, err := NewProxy(upstream.URL, allowedPaths, providers.GithubProviderKind, tt.secret, tt.ignoredUsers, tt.allowedUsers)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			p.Handler().ServeHTTP(rr, tt.request)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body %q)", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if tt.wantBody != "" && rr.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rr.Body.String(), tt.wantBody)
			}
			if got := len(upstream.requests()) > 0; got != tt.wantForwarded {
				t.Errorf("forwarded = %v, want %v", got, tt.wantForwarded)
			}
		})
	}
}

func TestProxyRequest_ForwardsRequest(t *testing.T) {
	t.Parallel()
	const (
		secret = "github-secret"
		body   = `{"ref":"refs/heads/main","sender":{"login":"octocat"}}`
	)
	upstream := newUpstreamWithHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = io.WriteString(w, "queued")
	})
	p, err := NewProxy(upstream.URL, []string{}, providers.GithubProviderKind, secret, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	req := newGithubRequest("/github-webhook/?token=abc&x=1", "push", githubSignature(secret, body), body)
	req.Header.Set(providers.XHubSignature256, githubSignature256(secret, body))
	req.Header.Set("Authorization", "Bearer client-token")
	req.Header.Set("X-Custom", "value")

	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted || rr.Body.String() != "queued" {
		t.Fatalf("response = %d %q, want %d %q", rr.Code, rr.Body.String(), http.StatusAccepted, "queued")
	}

	got := upstream.requests()
	if len(got) != 1 {
		t.Fatalf("upstream received %d requests, want 1", len(got))
	}
	fwd := got[0]
	if fwd.method != http.MethodPost {
		t.Errorf("method = %s, want POST", fwd.method)
	}
	if fwd.path != "/github-webhook/" {
		t.Errorf("path = %q, want /github-webhook/", fwd.path)
	}
	if fwd.rawQuery != "token=abc&x=1" {
		t.Errorf("query = %q, want token=abc&x=1", fwd.rawQuery)
	}
	if !bytes.Equal(fwd.body, []byte(body)) {
		t.Errorf("body = %q, want %q", fwd.body, body)
	}
	for _, h := range []string{providers.XHubSignature256, providers.XHubSignature, providers.XGitHubEvent, providers.XGitHubDelivery, providers.ContentTypeHeader} {
		if fwd.header.Get(h) != req.Header.Get(h) {
			t.Errorf("header %s = %q, want %q", h, fwd.header.Get(h), req.Header.Get(h))
		}
	}
	// Only the provider's known headers are forwarded.
	for _, h := range []string{"Authorization", "X-Custom"} {
		if v := fwd.header.Get(h); v != "" {
			t.Errorf("header %s forwarded with value %q, want it dropped", h, v)
		}
	}
}

func TestProxyRequest_ForwardsGitlabToken(t *testing.T) {
	t.Parallel()
	upstream := newUpstream(t)
	p, err := NewProxy(upstream.URL, []string{}, providers.GitlabProviderKind, proxyGitlabTestSecret, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	req := createGitlabRequestWithPayload(http.MethodPost, "/project/hook", proxyGitlabTestSecret,
		string(providers.GitlabPushEvent), proxyGitlabTestPayload)
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rr.Code, rr.Body.String())
	}
	got := upstream.requests()
	if len(got) != 1 {
		t.Fatalf("upstream received %d requests, want 1", len(got))
	}
	if got[0].header.Get(providers.XGitlabToken) != proxyGitlabTestSecret {
		t.Errorf("X-Gitlab-Token = %q, want %q", got[0].header.Get(providers.XGitlabToken), proxyGitlabTestSecret)
	}
	if got[0].header.Get(providers.XGitlabEvent) != string(providers.GitlabPushEvent) {
		t.Errorf("X-Gitlab-Event = %q", got[0].header.Get(providers.XGitlabEvent))
	}
	if !bytes.Equal(got[0].body, proxyGitlabTestPayload) {
		t.Error("payload was not forwarded unchanged")
	}
}

func TestProxyRequest_UpstreamErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		upstreamURL  string
		wantStatus   int
		wantContains string
	}{
		{
			name: "upstream server error is passed through",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "boom", http.StatusInternalServerError)
			},
			wantStatus:   http.StatusInternalServerError,
			wantContains: "Upstream returned status: 500 Internal Server Error",
		},
		{
			name: "upstream not found is passed through",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			},
			wantStatus:   http.StatusNotFound,
			wantContains: "Upstream returned status: 404 Not Found",
		},
		{
			name:         "unreachable upstream",
			upstreamURL:  closedServerURL(t),
			wantStatus:   http.StatusBadGateway,
			wantContains: "Error forwarding request to upstream",
		},
		{
			name:         "unsupported upstream scheme",
			upstreamURL:  "ftp://127.0.0.1",
			wantStatus:   http.StatusBadGateway,
			wantContains: "Error forwarding request to upstream",
		},
		{
			name: "truncated upstream response body",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", "100")
				_, _ = io.WriteString(w, "short")
			},
			wantStatus:   http.StatusBadGateway,
			wantContains: "Error reading upstream response body",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			upstreamURL := tt.upstreamURL
			if tt.handler != nil {
				upstreamURL = newUpstreamWithHandler(t, tt.handler).URL
			}
			p, err := NewProxy(upstreamURL, []string{}, providers.GitlabProviderKind, "", nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			req := createGitlabRequest(http.MethodPost, "/hook", "", proxyGitlabTestEvent, proxyGitlabTestBody)
			rr := httptest.NewRecorder()
			p.Handler().ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if !strings.Contains(rr.Body.String(), tt.wantContains) {
				t.Errorf("body = %q, want it to contain %q", rr.Body.String(), tt.wantContains)
			}
		})
	}
}

// Upstream error responses must release their connection. If the response
// body is left open, every request needs a new connection.
func TestProxyRequest_ReusesUpstreamConnectionAfterError(t *testing.T) {
	t.Parallel()
	const requests = 5
	var newConns atomic.Int32
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			newConns.Add(1)
		}
	}
	srv.Start()
	defer srv.Close()

	p, err := NewProxy(srv.URL, []string{}, providers.GitlabProviderKind, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for range requests {
		rr := httptest.NewRecorder()
		p.Handler().ServeHTTP(rr, createGitlabRequest(http.MethodPost, "/hook", "", proxyGitlabTestEvent, proxyGitlabTestBody))
		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
		}
	}

	if n := newConns.Load(); n >= requests {
		t.Errorf("upstream saw %d new connections for %d requests, want connections to be reused", n, requests)
	}
}

// TLS tests mutate the shared transport and must not run in parallel.
func TestUpstreamTLSVerification(t *testing.T) {
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	defer upstream.Close()
	t.Cleanup(func() {
		SetInsecureSkipVerify(false)
		transport.CloseIdleConnections()
	})

	p, err := NewProxy(upstream.URL, []string{}, providers.GitlabProviderKind, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	hook := createGitlabHook("", proxyGitlabTestEvent, proxyGitlabTestBody, http.MethodPost)

	assertRejected := func(t *testing.T) {
		t.Helper()
		resp, err := p.redirect(hook, upstream.URL+"/hook")
		if err == nil {
			_ = resp.Body.Close()
			t.Fatal("self-signed upstream certificate was accepted")
		}
		var unknownAuthority x509.UnknownAuthorityError
		if !errors.As(err, &unknownAuthority) {
			t.Errorf("error = %v, want x509.UnknownAuthorityError", err)
		}

		rr := httptest.NewRecorder()
		p.Handler().ServeHTTP(rr, createGitlabRequest(http.MethodPost, "/hook", "", proxyGitlabTestEvent, proxyGitlabTestBody))
		if rr.Code != http.StatusBadGateway {
			t.Errorf("proxy status = %d, want %d", rr.Code, http.StatusBadGateway)
		}
	}

	t.Run("rejected by default", assertRejected)

	t.Run("accepted with insecureSkipVerify", func(t *testing.T) {
		SetInsecureSkipVerify(true)
		defer func() {
			SetInsecureSkipVerify(false)
			transport.CloseIdleConnections()
		}()

		resp, err := p.redirect(hook, upstream.URL+"/hook")
		if err != nil {
			t.Fatalf("redirect() error = %v", err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want 200", resp.StatusCode)
		}

		rr := httptest.NewRecorder()
		p.Handler().ServeHTTP(rr, createGitlabRequest(http.MethodPost, "/hook", "", proxyGitlabTestEvent, proxyGitlabTestBody))
		if rr.Code != http.StatusOK || rr.Body.String() != "ok" {
			t.Errorf("proxy response = %d %q, want 200 %q", rr.Code, rr.Body.String(), "ok")
		}
	})

	t.Run("rejected again after re-enabling verification", assertRejected)
}

func TestProxy_Handler(t *testing.T) {
	t.Parallel()
	upstream := newUpstream(t)
	p, err := NewProxy(upstream.URL, []string{}, providers.GitlabProviderKind, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := p.Handler()

	tests := []struct {
		name            string
		request         *http.Request
		wantStatus      int
		wantContentType string
	}{
		{name: "health", request: httptest.NewRequest(http.MethodGet, "/health", nil), wantStatus: http.StatusOK},
		{name: "asset", request: httptest.NewRequest(http.MethodGet, "/assets/gitwebhookproxy-round-100px.png", nil), wantStatus: http.StatusOK, wantContentType: "image/png"},
		{name: "missing asset", request: httptest.NewRequest(http.MethodGet, "/assets/missing.css", nil), wantStatus: http.StatusNotFound},
		{name: "get on proxy path", request: httptest.NewRequest(http.MethodGet, "/hook", nil), wantStatus: http.StatusMethodNotAllowed},
		{name: "put on proxy path", request: httptest.NewRequest(http.MethodPut, "/hook", nil), wantStatus: http.StatusMethodNotAllowed},
		{
			name:       "post is proxied",
			request:    createGitlabRequest(http.MethodPost, "/hook", "", proxyGitlabTestEvent, proxyGitlabTestBody),
			wantStatus: http.StatusOK,
		},
		{
			name:       "post to health is proxied",
			request:    createGitlabRequest(http.MethodPost, "/health", "", proxyGitlabTestEvent, proxyGitlabTestBody),
			wantStatus: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, tt.request)
			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if tt.wantContentType != "" && rr.Header().Get("Content-Type") != tt.wantContentType {
				t.Errorf("Content-Type = %q, want %q", rr.Header().Get("Content-Type"), tt.wantContentType)
			}
		})
	}
}

func TestProxy_Run(t *testing.T) {
	t.Parallel()
	p := &Proxy{provider: providers.GithubProviderKind, upstreamURL: dummyUpstreamURL, allowedPaths: []string{}}

	t.Run("empty listen address panics", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if recover() == nil {
				t.Error("Run(\"\") did not panic")
			}
		}()
		_ = p.Run("  ")
	})

	t.Run("invalid listen address", func(t *testing.T) {
		t.Parallel()
		if err := p.Run("127.0.0.1:-1"); err == nil {
			t.Error("Run() returned nil error for an invalid address")
		}
	})

	t.Run("address in use", func(t *testing.T) {
		t.Parallel()
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = l.Close() }()
		if err := p.Run(l.Addr().String()); err == nil {
			t.Error("Run() returned nil error for an address already in use")
		}
	})
}

func TestProxy_newServerTimeouts(t *testing.T) {
	t.Parallel()
	p := &Proxy{provider: providers.GithubProviderKind, upstreamURL: dummyUpstreamURL, allowedPaths: []string{}}
	srv := p.newServer(":8080")

	if srv.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080", srv.Addr)
	}
	if srv.Handler == nil {
		t.Error("Handler is nil")
	}
	for name, got := range map[string]time.Duration{
		"ReadHeaderTimeout": srv.ReadHeaderTimeout,
		"ReadTimeout":       srv.ReadTimeout,
		"WriteTimeout":      srv.WriteTimeout,
		"IdleTimeout":       srv.IdleTimeout,
	} {
		if got <= 0 {
			t.Errorf("%s = %v, want a positive timeout", name, got)
		}
	}
	if srv.WriteTimeout <= httpClient.Timeout {
		t.Errorf("WriteTimeout %v must exceed the upstream client timeout %v", srv.WriteTimeout, httpClient.Timeout)
	}
}

// The ignore response echoes a payload-supplied user name, so it must not be
// interpretable as HTML.
func TestProxyRequest_IgnoreResponseIsPlainText(t *testing.T) {
	t.Parallel()
	const body = `{"sender":{"login":"<script>alert(1)</script>"}}`
	upstream := newUpstream(t)
	p, err := NewProxy(upstream.URL, []string{}, providers.GithubProviderKind, "", []string{"<script>alert(1)</script>"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, newGithubRequest("/hook", "push", "", body))

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/plain; charset=utf-8", got)
	}
	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if len(upstream.requests()) != 0 {
		t.Error("ignored request was forwarded")
	}
}
