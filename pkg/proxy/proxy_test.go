package proxy

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jarcoal/httpmock"
	"github.com/schwichtgit/GitWebhookProxy/pkg/providers"
)

const (
	proxyGitlabTestSecret = "testSecret"
	proxyGitlabTestEvent  = "testEvent"
	proxyGitlabTestBody   = "testBody"
	dummyUpstreamURL      = "https://upstream.example.com"
)

var (
	proxyGitlabTestPayload = getGitlabPayload()
)

func getGitlabPayload() []byte {
	payload, _ := os.ReadFile("gitlab_test_payload.json")
	return payload
}

func TestProxy_isPathAllowed(t *testing.T) {
	type fields struct {
		provider     string
		upstreamURL  string
		allowedPaths []string
		secret       string
	}
	type args struct {
		path string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "isPathAllowedWithValidMultipleAllowedPaths",
			fields: fields{
				provider:     providers.GithubProviderKind,
				upstreamURL:  "https://dummyurl.com",
				allowedPaths: []string{"/path1", "/path2"},
				secret:       "secret",
			},
			args: args{
				path: "/path2",
			},
			want: true,
		},
		{
			name: "isPathAllowedWithValidOneAllowedPaths",
			fields: fields{
				provider:     providers.GithubProviderKind,
				upstreamURL:  "https://dummyurl.com",
				allowedPaths: []string{"/path1"},
				secret:       "secret",
			},
			args: args{
				path: "/path1",
			},
			want: true,
		},
		{
			name: "isPathAllowedWithInvalidPath",
			fields: fields{
				provider:     providers.GithubProviderKind,
				upstreamURL:  "https://dummyurl.com",
				allowedPaths: []string{"/path1", "/path2"},
				secret:       "secret",
			},
			args: args{
				path: "/path3",
			},
			want: false,
		},
		{
			name: "isPathAllowedWithEmtpyPathArg",
			fields: fields{
				provider:     providers.GithubProviderKind,
				upstreamURL:  "https://dummyurl.com",
				allowedPaths: []string{"/path1", "/path2"},
				secret:       "secret",
			},
			args: args{
				path: "",
			},
			want: false,
		},
		{
			name: "isPathAllowedWithAllPathsAllowedAndEmptyPathArg",
			fields: fields{
				provider:     providers.GithubProviderKind,
				upstreamURL:  "https://dummyurl.com",
				allowedPaths: []string{},
				secret:       "secret",
			},
			args: args{
				path: "",
			},
			want: true,
		},
		{
			name: "isPathAllowedWithAllPathsAllowedAndRootEmptyPathArg",
			fields: fields{
				provider:     providers.GithubProviderKind,
				upstreamURL:  "https://dummyurl.com",
				allowedPaths: []string{},
				secret:       "secret",
			},
			args: args{
				path: "/",
			},
			want: true,
		},
		{
			name: "isPathAllowedWithAllPathsAllowedAndNonEmptyPathArg",
			fields: fields{
				provider:     providers.GithubProviderKind,
				upstreamURL:  "https://dummyurl.com",
				allowedPaths: []string{},
				secret:       "secret",
			},
			args: args{
				path: "/path1",
			},
			want: true,
		},
		{
			name: "isPathAllowedWithSomePathsAllowedAndRootPathArg",
			fields: fields{
				provider:     providers.GithubProviderKind,
				upstreamURL:  "https://dummyurl.com",
				allowedPaths: []string{"/path1", "/path2"},
				secret:       "secret",
			},
			args: args{
				path: "/",
			},
			want: false,
		},
		{
			name: "isPathAllowedWithSomePathsAllowedAndSubPathArg",
			fields: fields{
				provider:     providers.GithubProviderKind,
				upstreamURL:  "https://dummyurl.com",
				allowedPaths: []string{"/path1", "/path4"},
				secret:       "secret",
			},
			args: args{
				path: "/path2/path3",
			},
			want: false,
		},
		{
			name: "isPathAllowedWithSubPathsAllowedAndSubPathArg",
			fields: fields{
				provider:     providers.GithubProviderKind,
				upstreamURL:  "https://dummyurl.com",
				allowedPaths: []string{"/path1", "/path2/path3"},
				secret:       "secret",
			},
			args: args{
				path: "/path2/path3",
			},
			want: true,
		},
		{
			name: "isPathAllowedWithSubPathsAllowedAndPathArg",
			fields: fields{
				provider:     providers.GithubProviderKind,
				upstreamURL:  "https://dummyurl.com",
				allowedPaths: []string{"/path1", "/path2/path3"},
				secret:       "secret",
			},
			args: args{
				path: "/path2",
			},
			want: false,
		},
		{
			name: "isPathAllowedWithAllowedPathTrailingSlashAndNotInArg",
			fields: fields{
				provider:     providers.GithubProviderKind,
				upstreamURL:  "https://dummyurl.com",
				allowedPaths: []string{"/path1", "/path2/"},
				secret:       "secret",
			},
			args: args{
				path: "/path2",
			},
			want: true,
		},
		{
			name: "isPathAllowedWithSimpleAllowedPathAndTrailingSlashInArg",
			fields: fields{
				provider:     providers.GithubProviderKind,
				upstreamURL:  "https://dummyurl.com",
				allowedPaths: []string{"/path1", "/path2"},
				secret:       "secret",
			},
			args: args{
				path: "/path2/",
			},
			want: true,
		},
		{
			name: "isPathAllowedWithSurroundingWhitespaceInAllowedPath",
			fields: fields{
				provider:     providers.GithubProviderKind,
				allowedPaths: []string{"/path1", " /path2 "},
			},
			args: args{
				path: "/path2",
			},
			want: true,
		},
		{
			// Allowed paths match at path segment boundaries only.
			name: "isPathAllowedRejectsSiblingWithSamePrefix",
			fields: fields{
				provider:     providers.GithubProviderKind,
				allowedPaths: []string{"/path1"},
			},
			args: args{
				path: "/path10",
			},
			want: false,
		},
	}
	t.Parallel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &Proxy{
				provider:     tt.fields.provider,
				upstreamURL:  tt.fields.upstreamURL,
				allowedPaths: tt.fields.allowedPaths,
				secret:       tt.fields.secret,
			}
			if got := p.isPathAllowed(tt.args.path); got != tt.want {
				t.Errorf("Proxy.isPathAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProxy_redirect(t *testing.T) {
	t.Parallel()
	upstream := newUpstream(t)
	upstreamHost := strings.TrimPrefix(upstream.URL, "http://")

	tests := []struct {
		name               string
		upstreamURL        string
		redirectURL        string
		hook               *providers.Hook
		wantStatusCode     int
		wantRedirectedHost string // Only Host not complete URL
		wantErr            bool
	}{
		{
			name:               "TestRedirectWithValidValues",
			upstreamURL:        upstream.URL,
			redirectURL:        upstream.URL + "/post",
			hook:               createGitlabHook(proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestBody, http.MethodPost),
			wantStatusCode:     http.StatusOK,
			wantRedirectedHost: upstreamHost,
		},
		{
			name:               "TestRedirectWithGetUpstream",
			upstreamURL:        upstream.URL,
			redirectURL:        upstream.URL + "/get",
			hook:               createGitlabHook(proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestBody, http.MethodPost),
			wantStatusCode:     http.StatusMethodNotAllowed,
			wantRedirectedHost: upstreamHost,
		},
		{
			name:               "TestRedirectWithEmptyPath",
			upstreamURL:        upstream.URL + "/post",
			redirectURL:        upstream.URL + "/post",
			hook:               createGitlabHook(proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestBody, http.MethodPost),
			wantStatusCode:     http.StatusOK,
			wantRedirectedHost: upstreamHost,
		},
		{
			name:        "TestRedirectWithNilHook",
			upstreamURL: upstream.URL,
			redirectURL: upstream.URL + "/post",
			hook:        nil,
			wantErr:     true,
		},
		{
			name:        "TestRedirectWithUnreachableUrl",
			upstreamURL: closedServerURL(t),
			redirectURL: closedServerURL(t) + "/post",
			hook:        createGitlabHook(proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestBody, http.MethodPost),
			wantErr:     true,
		},
		{
			name:        "TestRedirectWithInvalidUrlScheme",
			upstreamURL: "htttpsss://" + upstreamHost,
			redirectURL: "htttpsss://" + upstreamHost + "/post",
			hook:        createGitlabHook(proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestBody, http.MethodPost),
			wantErr:     true,
		},
		{
			name:        "TestRedirectWithUnparsableUrl",
			upstreamURL: "http://[::1",
			redirectURL: "http://[::1/post",
			hook:        createGitlabHook(proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestBody, http.MethodPost),
			wantErr:     true,
		},
		{
			name:        "TestRedirectWithInvalidMethod",
			upstreamURL: upstream.URL,
			redirectURL: upstream.URL + "/post",
			hook:        createGitlabHook(proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestBody, "BAD METHOD"),
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &Proxy{
				provider:     providers.GitlabProviderKind,
				upstreamURL:  tt.upstreamURL,
				allowedPaths: []string{},
				secret:       "dummy",
			}
			gotResp, gotErr := p.redirect(tt.hook, tt.redirectURL)
			if (gotErr != nil) != tt.wantErr {
				t.Fatalf("Proxy.redirect() error = %v, wantErr %v", gotErr, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			defer func() { _ = gotResp.Body.Close() }()

			if gotResp.StatusCode != tt.wantStatusCode {
				t.Errorf("Proxy.redirect() got StatusCode in response= %v, want %v",
					gotResp.StatusCode, tt.wantStatusCode)
			}
			if gotResp.Request.Host != tt.wantRedirectedHost {
				t.Errorf("Proxy.redirect() got Redirected Host in response= %v, want Redirected Host= %v",
					gotResp.Request.Host, tt.wantRedirectedHost)
			}
		})
	}
}

// A host:port upstream without scheme cannot be parsed, so the default-scheme
// path is exercised with a port-less host name served by httpmock.
func TestProxy_redirectWithUrlWithoutScheme(t *testing.T) {
	httpmock.ActivateNonDefault(httpClient)
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder(http.MethodPost, "http://upstream.example/post",
		httpmock.NewStringResponder(http.StatusOK, ""))

	p := &Proxy{provider: providers.GitlabProviderKind, upstreamURL: "upstream.example", allowedPaths: []string{}}
	resp, err := p.redirect(
		createGitlabHook(proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestBody, http.MethodPost),
		"upstream.example/post")
	if err != nil {
		t.Fatalf("Proxy.redirect() error = %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Proxy.redirect() StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if resp.Request.URL.Scheme != "http" || resp.Request.Host != "upstream.example" {
		t.Errorf("Proxy.redirect() requested %s, want http://upstream.example/post", resp.Request.URL)
	}
}

func createGitlabRequest(method string, path string, tokenHeader string,
	eventHeader string, body string) *http.Request {
	return createGitlabRequestWithPayload(method, path, tokenHeader, eventHeader, []byte(body))
}

func createGitlabRequestWithPayload(method string, path string, tokenHeader string,
	eventHeader string, body []byte) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Add(providers.XGitlabToken, tokenHeader)
	req.Header.Add(providers.XGitlabEvent, eventHeader)
	req.Header.Add(providers.ContentTypeHeader, providers.DefaultContentTypeHeaderValue)
	return req
}

func createRequestWithWrongHeadersKeys(method string, path string, tokenHeader string,
	eventHeader string, body string) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	req.Header.Add("X-Wrong-Token", tokenHeader)
	req.Header.Add("X-Wrong-Event", eventHeader)
	return req
}

func createRequestWithoutHeaders(method string, path string, body string) *http.Request {
	return httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
}

func TestProxy_proxyRequest(t *testing.T) {
	t.Parallel()
	type fields struct {
		provider     string
		allowedPaths []string
		secret       string
	}
	tests := []struct {
		name           string
		fields         fields
		upstreamURL    string // defaults to a local test upstream
		request        *http.Request
		wantStatusCode int
		wantForwarded  bool
	}{
		{
			name:           "TestProxyRequestWithValidValues",
			fields:         fields{provider: providers.GitlabProviderKind, allowedPaths: []string{}, secret: proxyGitlabTestSecret},
			request:        createGitlabRequestWithPayload(http.MethodPost, "/post", proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestPayload),
			wantStatusCode: http.StatusOK,
			wantForwarded:  true,
		},
		{
			name:           "TestProxyRequestWithoutConfiguringSecret",
			fields:         fields{provider: providers.GitlabProviderKind, allowedPaths: []string{}, secret: ""},
			request:        createGitlabRequestWithPayload(http.MethodPost, "/post", proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestPayload),
			wantStatusCode: http.StatusOK,
			wantForwarded:  true,
		},
		{
			name:           "TestProxyRequestWithoutSecretHearderInRequest",
			fields:         fields{provider: providers.GitlabProviderKind, allowedPaths: []string{}, secret: proxyGitlabTestSecret},
			request:        createGitlabRequestWithPayload(http.MethodPost, "/post", "", proxyGitlabTestEvent, proxyGitlabTestPayload),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "TestProxyRequestWithInvalidSecretInHeader",
			fields:         fields{provider: providers.GitlabProviderKind, allowedPaths: []string{}, secret: proxyGitlabTestSecret},
			request:        createGitlabRequest(http.MethodPost, "/post", "InvalidSecret", proxyGitlabTestEvent, proxyGitlabTestBody),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "TestProxyRequestWithEmptySecretInHeader",
			fields:         fields{provider: providers.GitlabProviderKind, allowedPaths: []string{}, secret: proxyGitlabTestSecret},
			request:        createGitlabRequest(http.MethodPost, "/post", "", proxyGitlabTestEvent, proxyGitlabTestBody),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "TestProxyRequestWithEmptyEventInHeader",
			fields:         fields{provider: providers.GitlabProviderKind, allowedPaths: []string{}, secret: proxyGitlabTestSecret},
			request:        createGitlabRequest(http.MethodPost, "/post", proxyGitlabTestSecret, "", proxyGitlabTestBody),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "TestProxyRequestWithWrongHeaderKeys",
			fields:         fields{provider: providers.GitlabProviderKind, allowedPaths: []string{}, secret: proxyGitlabTestSecret},
			request:        createRequestWithWrongHeadersKeys(http.MethodPost, "/post", proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestBody),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "TestProxyRequestWithoutHeaderKeys",
			fields:         fields{provider: providers.GitlabProviderKind, allowedPaths: []string{}, secret: proxyGitlabTestSecret},
			request:        createRequestWithoutHeaders(http.MethodPost, "/post", proxyGitlabTestBody),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "TestProxyRequestWithUnsupportedUrlPath",
			fields:         fields{provider: providers.GitlabProviderKind, allowedPaths: []string{}, secret: proxyGitlabTestSecret},
			request:        createGitlabRequestWithPayload(http.MethodPost, "/get", proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestPayload),
			wantStatusCode: http.StatusMethodNotAllowed,
			wantForwarded:  true,
		},
		{
			name:           "TestProxyRequestWithInvalidHttpMethod",
			fields:         fields{provider: providers.GitlabProviderKind, allowedPaths: []string{}, secret: proxyGitlabTestSecret},
			request:        createGitlabRequest(http.MethodGet, "/post", proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestBody),
			wantStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:           "TestProxyRequestWithEmptyBody",
			fields:         fields{provider: providers.GitlabProviderKind, allowedPaths: []string{}, secret: proxyGitlabTestSecret},
			request:        createGitlabRequest(http.MethodPost, "/post", proxyGitlabTestSecret, proxyGitlabTestEvent, ""),
			wantStatusCode: http.StatusOK,
			wantForwarded:  true,
		},
		{
			name:           "TestProxyRequestWithNotAllowedPath",
			fields:         fields{provider: providers.GitlabProviderKind, allowedPaths: []string{"/path1"}, secret: proxyGitlabTestSecret},
			request:        createGitlabRequest(http.MethodPost, "/post", proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestSecret),
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "TestProxyRequestWithAllowedPath",
			fields:         fields{provider: providers.GitlabProviderKind, allowedPaths: []string{"/post"}, secret: proxyGitlabTestSecret},
			request:        createGitlabRequest(http.MethodPost, "/post", proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestSecret),
			wantStatusCode: http.StatusOK,
			wantForwarded:  true,
		},
		{
			name:           "TestProxyRequestWithInvalidUpstreamUrl",
			fields:         fields{provider: providers.GitlabProviderKind, allowedPaths: []string{}, secret: proxyGitlabTestSecret},
			upstreamURL:    closedServerURL(t),
			request:        createGitlabRequest(http.MethodPost, "/post", proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestSecret),
			wantStatusCode: http.StatusBadGateway,
		},
		{
			name:           "TestProxyRequestWithInvalidProvider",
			fields:         fields{provider: "invalid", allowedPaths: []string{}, secret: proxyGitlabTestSecret},
			request:        createGitlabRequest(http.MethodPost, "/post", proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestSecret),
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:           "TestProxyRequestWithWrongProviderKind",
			fields:         fields{provider: providers.GithubProviderKind, allowedPaths: []string{}, secret: proxyGitlabTestSecret},
			request:        createGitlabRequest(http.MethodPost, "/post", proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestSecret),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "TestProxyRequestWithInvalidSecretInProvider",
			fields:         fields{provider: providers.GitlabProviderKind, allowedPaths: []string{}, secret: "wrong"},
			request:        createGitlabRequest(http.MethodPost, "/post", proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestSecret),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "TestProxyRequestWithEmptySecretInProvider",
			fields:         fields{provider: providers.GitlabProviderKind, allowedPaths: []string{}, secret: ""},
			request:        createGitlabRequest(http.MethodPost, "/post", proxyGitlabTestSecret, proxyGitlabTestEvent, proxyGitlabTestSecret),
			wantStatusCode: http.StatusOK,
			wantForwarded:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			upstream := newUpstream(t)
			upstreamURL := tt.upstreamURL
			if upstreamURL == "" {
				upstreamURL = upstream.URL
			}
			p := &Proxy{
				provider:     tt.fields.provider,
				upstreamURL:  upstreamURL,
				allowedPaths: tt.fields.allowedPaths,
				secret:       tt.fields.secret,
			}
			router := chi.NewRouter()
			router.Post("/*", p.proxyRequest)

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, tt.request)

			if status := rr.Code; status != tt.wantStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.wantStatusCode)
			}
			if got := len(upstream.requests()) > 0; got != tt.wantForwarded {
				t.Errorf("request forwarded to upstream = %v, want %v", got, tt.wantForwarded)
			}
		})
	}
}

func TestProxy_health(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		httpMethod     string
		wantStatusCode int
		wantBody       string
	}{
		{
			name:           "TestHealthCheckGet",
			httpMethod:     http.MethodGet,
			wantStatusCode: http.StatusOK,
			wantBody:       "I'm Healthy and I know it! ;) ",
		},
		{
			name:           "TestHealthCheckPost",
			httpMethod:     http.MethodPost,
			wantStatusCode: http.StatusMethodNotAllowed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &Proxy{}
			router := chi.NewRouter()
			router.Get("/health", p.health)

			req := httptest.NewRequest(tt.httpMethod, "/health", nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.wantStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.wantStatusCode)
			}
			if tt.wantBody != "" && rr.Body.String() != tt.wantBody {
				t.Errorf("handler returned body %q, want %q", rr.Body.String(), tt.wantBody)
			}
		})
	}
}

func createGitlabHook(tokenHeader string, tokenEvent string, body string, method string) *providers.Hook {
	return &providers.Hook{
		Headers: map[string]string{
			providers.XGitlabToken: tokenHeader,
			providers.XGitlabEvent: tokenEvent,
		},
		Payload:       []byte(body),
		RequestMethod: method,
	}
}

func TestNewProxy(t *testing.T) {
	type args struct {
		upstreamURL  string
		allowedPaths []string
		provider     string
		secret       string
		ignoredUsers []string
	}
	tests := []struct {
		name    string
		args    args
		want    *Proxy
		wantErr bool
	}{
		{
			name: "TestNewProxyWithValidArgs",
			args: args{
				upstreamURL:  dummyUpstreamURL,
				allowedPaths: []string{},
				provider:     providers.GitlabProviderKind,
				secret:       proxyGitlabTestSecret,
			},
			want: &Proxy{
				upstreamURL:  dummyUpstreamURL,
				allowedPaths: []string{},
				provider:     providers.GitlabProviderKind,
				secret:       proxyGitlabTestSecret,
			},
		},
		{
			name: "TestNewProxyWithEmptyUpstreamURL",
			args: args{
				upstreamURL:  "",
				allowedPaths: []string{},
				provider:     providers.GitlabProviderKind,
				secret:       proxyGitlabTestSecret,
			},
			wantErr: true,
		},
		{
			name: "TestNewProxyWithNilAllowedPaths",
			args: args{
				upstreamURL:  dummyUpstreamURL,
				allowedPaths: nil,
				provider:     providers.GitlabProviderKind,
				secret:       proxyGitlabTestSecret,
			},
			wantErr: true,
		},
		{
			name: "TestNewProxyWithEmptyProvider",
			args: args{
				upstreamURL:  dummyUpstreamURL,
				allowedPaths: []string{},
				provider:     "",
				secret:       proxyGitlabTestSecret,
			},
			wantErr: true,
		},
		{
			name: "TestNewProxyWithEmptySecret",
			args: args{
				upstreamURL:  dummyUpstreamURL,
				allowedPaths: nil,
				provider:     providers.GitlabProviderKind,
				secret:       "",
			},
			wantErr: true,
		},
		{
			name: "TestNewProxyWithValidArgsAndAllowedPaths",
			args: args{
				upstreamURL:  dummyUpstreamURL,
				allowedPaths: []string{"/path1", "/path2"},
				provider:     providers.GitlabProviderKind,
				secret:       proxyGitlabTestSecret,
				ignoredUsers: []string{"user1"},
			},
			want: &Proxy{
				upstreamURL:  dummyUpstreamURL,
				allowedPaths: []string{"/path1", "/path2"},
				provider:     providers.GitlabProviderKind,
				secret:       proxyGitlabTestSecret,
				ignoredUsers: []string{"user1"},
			},
		},
	}
	t.Parallel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewProxy(tt.args.upstreamURL, tt.args.allowedPaths, tt.args.provider, tt.args.secret, tt.args.ignoredUsers, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewProxy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewProxy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProxy_isIgnoredUser(t *testing.T) {
	type fields struct {
		provider     string
		upstreamURL  string
		allowedPaths []string
		secret       string
		ignoredUsers []string
	}
	type args struct {
		committer string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "TestIsIgnoredUserWithEmptyList",
			fields: fields{
				provider:     providers.GithubProviderKind,
				upstreamURL:  "https://dummyurl.com",
				allowedPaths: []string{"/path1", "/path2"},
				secret:       "secret",
				ignoredUsers: []string{},
			},
			args: args{
				committer: "user",
			},
			want: false,
		},
		{
			name: "TestIsIgnoredUserWithValidList",
			fields: fields{
				provider:     providers.GithubProviderKind,
				upstreamURL:  "https://dummyurl.com",
				allowedPaths: []string{"/path1", "/path2"},
				secret:       "secret",
				ignoredUsers: []string{"user1", "user2"},
			},
			args: args{
				committer: "user2",
			},
			want: true,
		},
		{
			name:   "TestIsIgnoredUserWithUserNotInList",
			fields: fields{provider: providers.GithubProviderKind, ignoredUsers: []string{"user1", "user2"}},
			args:   args{committer: "user3"},
			want:   false,
		},
		{
			name:   "TestIsIgnoredUserMatchIsCaseSensitive",
			fields: fields{provider: providers.GithubProviderKind, ignoredUsers: []string{"user1"}},
			args:   args{committer: "User1"},
			want:   false,
		},
		{
			name:   "TestIsIgnoredUserWithEmptyGithubCommitter",
			fields: fields{provider: providers.GithubProviderKind, ignoredUsers: []string{}},
			args:   args{committer: ""},
			want:   true,
		},
		{
			name:   "TestIsIgnoredUserWithEmptyGitlabCommitter",
			fields: fields{provider: providers.GitlabProviderKind, ignoredUsers: []string{}},
			args:   args{committer: ""},
			want:   false,
		},
	}
	t.Parallel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &Proxy{
				provider:     tt.fields.provider,
				upstreamURL:  tt.fields.upstreamURL,
				allowedPaths: tt.fields.allowedPaths,
				secret:       tt.fields.secret,
				ignoredUsers: tt.fields.ignoredUsers,
			}
			if got := p.isIgnoredUser(tt.args.committer); got != tt.want {
				t.Errorf("Proxy.isIgnoredUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProxy_isAllowedUser(t *testing.T) {
	type fields struct {
		provider     string
		upstreamURL  string
		allowedPaths []string
		secret       string
		allowedUsers []string
	}
	type args struct {
		committer string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "TestIsAllowedUserWithEmptyList",
			fields: fields{
				provider:     providers.GithubProviderKind,
				upstreamURL:  "https://dummyurl.com",
				allowedPaths: []string{"/path1", "/path2"},
				secret:       "secret",
				allowedUsers: []string{},
			},
			args: args{
				committer: "user",
			},
			want: true,
		},
		{
			name: "TestIsAllowedUserWithValidList",
			fields: fields{
				provider:     providers.GithubProviderKind,
				upstreamURL:  "https://dummyurl.com",
				allowedPaths: []string{"/path1", "/path2"},
				secret:       "secret",
				allowedUsers: []string{"user1", "user2"},
			},
			args: args{
				committer: "user2",
			},
			want: true,
		},
		{
			name: "TestIsNotAllowedUserWithValidList",
			fields: fields{
				provider:     providers.GithubProviderKind,
				upstreamURL:  "https://dummyurl.com",
				allowedPaths: []string{"/path1", "/path2"},
				secret:       "secret",
				allowedUsers: []string{"user1", "user2"},
			},
			args: args{
				committer: "user3",
			},
			want: false,
		},
	}
	t.Parallel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &Proxy{
				provider:     tt.fields.provider,
				upstreamURL:  tt.fields.upstreamURL,
				allowedPaths: tt.fields.allowedPaths,
				secret:       tt.fields.secret,
				allowedUsers: tt.fields.allowedUsers,
			}
			if got := p.isAllowedUser(tt.args.committer); got != tt.want {
				t.Errorf("Proxy.isAllowedUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultTLSVerificationEnabled(t *testing.T) {
	if transport.TLSClientConfig == nil {
		t.Fatal("transport must have an explicit TLS configuration")
	}
	if transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("upstream TLS verification must be enabled by default (InsecureSkipVerify should be false)")
	}
}

func TestSetInsecureSkipVerify(t *testing.T) {
	// Restore the secure default after the test so other tests are unaffected.
	defer SetInsecureSkipVerify(false)

	SetInsecureSkipVerify(true)
	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("SetInsecureSkipVerify(true) should disable TLS verification")
	}

	SetInsecureSkipVerify(false)
	if transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("SetInsecureSkipVerify(false) should re-enable TLS verification")
	}
}
