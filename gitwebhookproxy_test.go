package main

import (
	"bytes"
	"crypto/fips140"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/schwichtgit/GitWebhookProxy/pkg/proxy"
	"github.com/spf13/pflag"
)

func TestParseConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		envVars []string
		args    []string
		want    *config
		wantErr error
		wantLog string
	}{
		{
			name:    "env var uppercase",
			envVars: []string{"GWP_UPSTREAMURL=https://jenkins.example.com"},
			want:    defaultConfig("https://jenkins.example.com"),
		},
		{
			name:    "env var mixed case",
			envVars: []string{"GWP_upstreamURL=https://jenkins.example.com"},
			want:    defaultConfig("https://jenkins.example.com"),
		},
		{
			name:    "command line flag takes precedence",
			envVars: []string{"GWP_UPSTREAMURL=https://jenkins.example.com"},
			args:    []string{"--upstreamURL=https://jenkins2.example.com"},
			want:    defaultConfig("https://jenkins2.example.com"),
		},
		{
			name: "multiple env vars",
			envVars: []string{
				"GWP_UPSTREAMURL=https://jenkins.example.com",
				"GWP_ALLOWEDPATHS=/path1,/path2",
				"GWP_PROVIDER=gitlab",
			},
			want: func() *config {
				c := defaultConfig("https://jenkins.example.com")
				c.allowedPaths = []string{"/path1", "/path2"}
				c.provider = "gitlab"
				return c
			}(),
		},
		{
			name: "all env vars",
			envVars: []string{
				"GWP_UPSTREAMURL=https://jenkins.example.com",
				"GWP_LISTEN=:9090",
				"GWP_SECRET=s3cret",
				"GWP_PROVIDER=GitLab",
				"GWP_ALLOWEDPATHS=/a",
				"GWP_IGNOREDUSERS=bot1,bot2",
				"GWP_ALLOWEDUSER=alice",
				"GWP_INSECURESKIPVERIFY=true",
			},
			want: &config{
				listenAddress:      ":9090",
				upstreamURL:        "https://jenkins.example.com",
				secret:             "s3cret",
				provider:           "gitlab",
				allowedPaths:       []string{"/a"},
				ignoredUsers:       []string{"bot1", "bot2"},
				allowedUsers:       []string{"alice"},
				insecureSkipVerify: true,
			},
		},
		{
			name: "all flags",
			args: []string{
				"--upstreamURL=https://jenkins.example.com",
				"--listen=127.0.0.1:8081",
				"--secret=s3cret",
				"--provider=GITHUB",
				"--allowedPaths=/a,/b",
				"--ignoredUsers=bot",
				"--allowedUser=alice,bob",
				"--insecureSkipVerify",
			},
			want: &config{
				listenAddress:      "127.0.0.1:8081",
				upstreamURL:        "https://jenkins.example.com",
				secret:             "s3cret",
				provider:           "github",
				allowedPaths:       []string{"/a", "/b"},
				ignoredUsers:       []string{"bot"},
				allowedUsers:       []string{"alice", "bob"},
				insecureSkipVerify: true,
			},
		},
		{
			name:    "flag overrides only its own env var",
			envVars: []string{"GWP_UPSTREAMURL=https://env.example.com", "GWP_SECRET=from-env"},
			args:    []string{"--secret=from-flag"},
			want: func() *config {
				c := defaultConfig("https://env.example.com")
				c.secret = "from-flag"
				return c
			}(),
		},
		{
			name: "unrelated and malformed env entries are ignored",
			envVars: []string{
				"UPSTREAMURL=https://ignored.example.com",
				"GWP_UNKNOWN=value",
				"GWP_NOVALUE",
				"GWP_UPSTREAMURL=https://jenkins.example.com",
			},
			want: defaultConfig("https://jenkins.example.com"),
		},
		{
			name:    "env value containing equals sign",
			envVars: []string{"GWP_UPSTREAMURL=https://jenkins.example.com/hook?a=b"},
			want:    defaultConfig("https://jenkins.example.com/hook?a=b"),
		},
		{
			name:    "invalid env value is logged and ignored",
			envVars: []string{"GWP_UPSTREAMURL=https://jenkins.example.com", "GWP_INSECURESKIPVERIFY=maybe"},
			want:    defaultConfig("https://jenkins.example.com"),
			wantLog: "Error setting flag insecureSkipVerify",
		},
		{
			name: "unknown flags are ignored",
			args: []string{"--upstreamURL=https://jenkins.example.com", "--unknown=1"},
			want: defaultConfig("https://jenkins.example.com"),
		},
		{
			name:    "no url specified",
			wantErr: errUsage,
			wantLog: "Required flag 'upstreamURL' not specified",
		},
		{
			name:    "blank url",
			args:    []string{"--upstreamURL=  "},
			wantErr: errUsage,
			wantLog: "Required flag 'upstreamURL' not specified",
		},
		{
			name:    "invalid flag value",
			args:    []string{"--upstreamURL=https://jenkins.example.com", "--insecureSkipVerify=maybe"},
			wantErr: errUsage,
		},
		{
			name:    "help",
			args:    []string{"--help"},
			wantErr: pflag.ErrHelp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr, logs bytes.Buffer
			logger := log.New(&logs, "", 0)

			got, err := parseConfig("gitwebhookproxy", tt.args, tt.envVars, &stdout, &stderr, logger)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("parseConfig() error = %v, want %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseConfig() = %+v, want %+v", got, tt.want)
			}
			if !strings.Contains(logs.String(), tt.wantLog) {
				t.Errorf("log output %q does not contain %q", logs.String(), tt.wantLog)
			}
			if tt.wantErr != nil && !strings.Contains(stderr.String(), "Usage of gitwebhookproxy:") {
				t.Errorf("usage text not printed, stderr = %q", stderr.String())
			}
		})
	}
}

func defaultConfig(upstreamURL string) *config {
	return &config{
		listenAddress: ":8080",
		upstreamURL:   upstreamURL,
		provider:      "github",
		allowedPaths:  []string{},
		ignoredUsers:  []string{},
		allowedUsers:  []string{},
	}
}

func TestSplitList(t *testing.T) {
	t.Parallel()
	tests := map[string][]string{
		"":      {},
		"a":     {"a"},
		"a,b":   {"a", "b"},
		"a,,b,": {"a", "", "b", ""},
		" a, b": {" a", " b"},
	}
	for in, want := range tests {
		if got := splitList(in); !reflect.DeepEqual(got, want) {
			t.Errorf("splitList(%q) = %q, want %q", in, got, want)
		}
	}
}

// run changes the proxy package's TLS setting, so these tests are not parallel.
func TestRun(t *testing.T) {
	t.Cleanup(func() { proxy.SetInsecureSkipVerify(false) })

	errServe := errors.New("serve failed")
	tests := []struct {
		name         string
		args         []string
		environ      []string
		serveErr     error
		wantErr      error
		wantServe    bool
		wantListen   string
		wantLogs     []string
		wantNotLogs  []string
		wantStdout   string
		wantUsageErr bool
	}{
		{
			name:       "starts with defaults",
			args:       []string{"gitwebhookproxy", "--upstreamURL=http://jenkins:8080"},
			wantServe:  true,
			wantListen: ":8080",
			wantLogs: []string{
				"Git WebHook Proxy dev started with provider 'github'",
			},
			wantNotLogs: []string{"WARNING: upstream TLS certificate verification is disabled"},
		},
		{
			name:       "configured from environment",
			args:       []string{"gitwebhookproxy"},
			environ:    []string{"GWP_UPSTREAMURL=http://gitlab-ci:8080", "GWP_PROVIDER=GitLab", "GWP_LISTEN=:9000"},
			wantServe:  true,
			wantListen: ":9000",
			wantLogs:   []string{"started with provider 'gitlab'"},
		},
		{
			name:       "insecureSkipVerify logs a warning",
			args:       []string{"gitwebhookproxy", "--upstreamURL=https://jenkins", "--insecureSkipVerify"},
			wantServe:  true,
			wantListen: ":8080",
			wantLogs:   []string{"WARNING: upstream TLS certificate verification is disabled (insecureSkipVerify=true)"},
		},
		{
			name:     "serve error is returned",
			args:     []string{"gitwebhookproxy", "--upstreamURL=http://jenkins"},
			serveErr: errServe,
			wantErr:  errServe,
		},
		{
			name:         "missing upstream url",
			args:         []string{"gitwebhookproxy"},
			wantErr:      errUsage,
			wantLogs:     []string{"Required flag 'upstreamURL' not specified"},
			wantStdout:   "\n\n",
			wantUsageErr: true,
		},
		{
			name:    "invalid provider is rejected by the proxy",
			args:    []string{"gitwebhookproxy", "--upstreamURL=http://jenkins", "--provider= "},
			wantErr: errors.New("cannot create proxy with empty provider"),
		},
		{
			name:       "no arguments at all",
			args:       nil,
			environ:    []string{"GWP_UPSTREAMURL=http://jenkins"},
			wantServe:  true,
			wantListen: ":8080",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			served := false
			var gotListen string
			serve := func(p *proxy.Proxy, listenAddress string) error {
				if p == nil {
					t.Error("serve called with nil proxy")
				}
				served = true
				gotListen = listenAddress
				return tt.serveErr
			}

			err := run(tt.args, tt.environ, &stdout, &stderr, serve)
			switch {
			case tt.wantErr == nil && err != nil:
				t.Fatalf("run() error = %v", err)
			case tt.wantErr != nil && (err == nil || (!errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error())):
				t.Fatalf("run() error = %v, want %v", err, tt.wantErr)
			}
			if served != tt.wantServe && tt.serveErr == nil {
				t.Errorf("serve called = %v, want %v", served, tt.wantServe)
			}
			if tt.wantServe && gotListen != tt.wantListen {
				t.Errorf("listen address = %q, want %q", gotListen, tt.wantListen)
			}

			logs := stderr.String()
			for _, want := range tt.wantLogs {
				if !strings.Contains(logs, want) {
					t.Errorf("stderr %q does not contain %q", logs, want)
				}
			}
			for _, notWant := range tt.wantNotLogs {
				if strings.Contains(logs, notWant) {
					t.Errorf("stderr %q unexpectedly contains %q", logs, notWant)
				}
			}
			if tt.wantServe {
				fips := "FIPS 140-3 mode disabled"
				if fips140.Enabled() {
					fips = "FIPS 140-3 mode enabled (Go Cryptographic Module " + fips140.Version() + ")"
				}
				if !strings.Contains(logs, fips) {
					t.Errorf("stderr %q does not report FIPS status %q", logs, fips)
				}
			}
			if tt.wantUsageErr {
				if !strings.Contains(logs, "Usage of gitwebhookproxy:") || !strings.Contains(logs, "--upstreamURL string") {
					t.Errorf("usage text missing from stderr: %q", logs)
				}
				if stdout.String() != tt.wantStdout {
					t.Errorf("stdout = %q, want %q", stdout.String(), tt.wantStdout)
				}
			}
		})
	}
}

// The insecureSkipVerify flag must reach the proxy's upstream transport: a
// self-signed upstream is rejected by default and accepted with the flag.
func TestRunAppliesInsecureSkipVerify(t *testing.T) {
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	defer upstream.Close()
	t.Cleanup(func() { proxy.SetInsecureSkipVerify(false) })

	tests := []struct {
		name       string
		flags      []string
		wantStatus int
	}{
		{name: "verification on by default", wantStatus: http.StatusBadGateway},
		{name: "insecureSkipVerify", flags: []string{"--insecureSkipVerify"}, wantStatus: http.StatusOK},
		{name: "insecureSkipVerify=false", flags: []string{"--insecureSkipVerify=false"}, wantStatus: http.StatusBadGateway},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A fresh server per case avoids reusing a connection that was
			// established under a different verification setting.
			upstream := httptest.NewTLSServer(upstream.Config.Handler)
			defer upstream.Close()

			args := append([]string{"gitwebhookproxy", "--upstreamURL=" + upstream.URL, "--provider=gitlab"}, tt.flags...)
			serve := func(p *proxy.Proxy, _ string) error {
				req := httptest.NewRequest(http.MethodPost, "/hook", strings.NewReader("{}"))
				req.Header.Set("X-Gitlab-Event", "Push Hook")
				req.Header.Set("Content-Type", "application/json")
				rr := httptest.NewRecorder()
				p.Handler().ServeHTTP(rr, req)
				if rr.Code != tt.wantStatus {
					t.Errorf("status = %d, want %d (body %q)", rr.Code, tt.wantStatus, rr.Body.String())
				}
				return nil
			}
			if err := run(args, nil, io.Discard, io.Discard, serve); err != nil {
				t.Fatalf("run() error = %v", err)
			}
		})
	}
}
