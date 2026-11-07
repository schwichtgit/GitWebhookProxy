package parser

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stakater/GitWebhookProxy/pkg/providers"
)

const (
	parserGitlabTestSecret = "testSecret"
	parserGitlabTestEvent  = "testEvent"
	parserGitlabTestBody   = "testBody"
)

func createGitlabRequest(method string, path string, tokenHeader string,
	eventHeader string, body string) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	req.Header.Add(providers.XGitlabToken, tokenHeader)
	req.Header.Add(providers.XGitlabEvent, eventHeader)
	req.Header.Add(providers.ContentTypeHeader, providers.DefaultContentTypeHeaderValue)
	return req
}

func createRequestWithWrongHeaders(method string, path string, tokenHeader string,
	eventHeader string, body string) *http.Request {
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	req.Header.Add("X-Wrong-Token", tokenHeader)
	req.Header.Add("X-Wrong-Event", eventHeader)
	return req
}

func createGitlabProvider(secret string) providers.Provider {
	provider, _ := providers.NewGitlabProvider(secret)
	return provider
}

func createGitlabHook(tokenHeader string, tokenEvent string, body string, method string) *providers.Hook {
	return &providers.Hook{
		Headers: map[string]string{
			providers.XGitlabToken:      tokenHeader,
			providers.XGitlabEvent:      tokenEvent,
			providers.ContentTypeHeader: providers.DefaultContentTypeHeaderValue,
		},
		Payload:       []byte(body),
		RequestMethod: method,
	}
}

func TestParse(t *testing.T) {
	type args struct {
		req      *http.Request
		provider providers.Provider
	}

	tests := []struct {
		name    string
		args    args
		want    *providers.Hook
		wantErr bool
	}{
		{
			name: "TestParseWithCorrectRequestValues",
			args: args{
				req: createGitlabRequest(http.MethodPost, "/dummy", parserGitlabTestSecret,
					parserGitlabTestEvent, parserGitlabTestBody),
				provider: createGitlabProvider(parserGitlabTestSecret),
			},
			want: createGitlabHook(parserGitlabTestSecret, parserGitlabTestEvent, parserGitlabTestBody, http.MethodPost),
		},
		{
			name: "TestParseWithEmptyTokenHeaderValue",
			args: args{
				req: createGitlabRequest(http.MethodPost, "/dummy", "",
					parserGitlabTestEvent, parserGitlabTestBody),
				provider: createGitlabProvider(parserGitlabTestSecret),
			},
			wantErr: true,
		},
		{
			name: "TestParseWithNoEventHeaderValue",
			args: args{
				req: createGitlabRequest(http.MethodPost, "/dummy", parserGitlabTestSecret,
					"", parserGitlabTestBody),
				provider: createGitlabProvider(parserGitlabTestSecret),
			},
			wantErr: true,
		},
		{
			name: "TestParseWithNoBody",
			args: args{
				req: createGitlabRequest(http.MethodPost, "/dummy", parserGitlabTestSecret,
					parserGitlabTestEvent, ""),
				provider: createGitlabProvider(parserGitlabTestSecret),
			},
			want: createGitlabHook(parserGitlabTestSecret, parserGitlabTestEvent, "", http.MethodPost),
		},
		{
			name: "TestParseWithNoHeaders",
			args: args{
				req:      httptest.NewRequest(http.MethodPost, "/dummy", bytes.NewReader([]byte(parserGitlabTestBody))),
				provider: createGitlabProvider(parserGitlabTestSecret),
			},
			wantErr: true,
		},
		{
			name: "TestParseWithWrongHeaderKeys",
			args: args{
				req: createRequestWithWrongHeaders(http.MethodPost, "/dummy", parserGitlabTestSecret,
					parserGitlabTestEvent, parserGitlabTestBody),
				provider: createGitlabProvider(parserGitlabTestSecret),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.args.req, tt.args.provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestParseBodyReadError(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/dummy", failingReader{})
	req.Header.Set(providers.XGitlabEvent, parserGitlabTestEvent)
	req.Header.Set(providers.ContentTypeHeader, providers.DefaultContentTypeHeaderValue)

	hook, err := Parse(req, createGitlabProvider(""))
	if err == nil {
		t.Fatalf("Parse() = %v, want error", hook)
	}
}

func TestParseOptionalHeaders(t *testing.T) {
	t.Parallel()
	github, err := providers.NewGithubProvider("secret")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		headers map[string]string
		want    map[string]string
	}{
		{
			name:    "no signature headers",
			headers: map[string]string{},
			want:    map[string]string{},
		},
		{
			name:    "sha256 only",
			headers: map[string]string{providers.XHubSignature256: "sha256=abc"},
			want:    map[string]string{providers.XHubSignature256: "sha256=abc"},
		},
		{
			name: "both signatures",
			headers: map[string]string{
				providers.XHubSignature256: "sha256=abc",
				providers.XHubSignature:    "sha1=def",
			},
			want: map[string]string{
				providers.XHubSignature256: "sha256=abc",
				providers.XHubSignature:    "sha1=def",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/dummy", bytes.NewReader(nil))
			req.Header.Set(providers.XGitHubDelivery, "delivery")
			req.Header.Set(providers.XGitHubEvent, "push")
			req.Header.Set(providers.ContentTypeHeader, providers.DefaultContentTypeHeaderValue)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			hook, err := Parse(req, github)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			for _, k := range []string{providers.XHubSignature256, providers.XHubSignature} {
				got, ok := hook.Headers[k]
				want, wantOK := tt.want[k]
				if ok != wantOK || got != want {
					t.Errorf("hook header %s = %q (present %v), want %q (present %v)", k, got, ok, want, wantOK)
				}
			}
		})
	}
}
