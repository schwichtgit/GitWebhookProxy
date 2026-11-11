package providers

import (
	"reflect"
	"strings"
	"testing"
)

const (
	githubTestSecret = "MyGithubTestSecret"
)

func TestNewGithubProvider(t *testing.T) {
	type args struct {
		secret string
	}
	tests := []struct {
		name    string
		args    args
		want    *GithubProvider
		wantErr bool
	}{
		{
			name: "TestNewGithubProviderWithCorrectSecret",
			args: args{
				secret: githubTestSecret,
			},
			want: &GithubProvider{
				secret: githubTestSecret,
			},
			wantErr: false,
		},
		{
			name: "TestNewGithubProviderWithEmptySecret",
			args: args{
				secret: "",
			},
			want: &GithubProvider{
				secret: "",
			},
			wantErr: false,
		},
		{
			name:    "TestNewGithubProviderWithNoSecret",
			args:    args{},
			want:    &GithubProvider{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGithubProvider(tt.args.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGithubProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewGithubProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGithubProvider_GetHeaderKeys(t *testing.T) {
	type fields struct {
		secret string
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "TestGetHeaderKeysWithCorrectValues",
			want: []string{XGitHubDelivery, XGitHubEvent, ContentTypeHeader},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &GithubProvider{
				secret: tt.fields.secret,
			}
			if got := p.GetHeaderKeys(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GithubProvider.GetHeaderKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGithubProvider_HeaderKeysWithSecret(t *testing.T) {
	t.Parallel()
	p := &GithubProvider{secret: githubTestSecret}
	wantRequired := []string{XGitHubDelivery, XGitHubEvent, ContentTypeHeader}
	if got := p.GetHeaderKeys(); !reflect.DeepEqual(got, wantRequired) {
		t.Errorf("GetHeaderKeys() = %v, want %v", got, wantRequired)
	}
	wantOptional := []string{XHubSignature256, XHubSignature}
	if got := p.GetOptionalHeaderKeys(); !reflect.DeepEqual(got, wantOptional) {
		t.Errorf("GetOptionalHeaderKeys() = %v, want %v", got, wantOptional)
	}
}

func TestGithubProvider_OptionalHeaderKeysWithoutSecret(t *testing.T) {
	t.Parallel()
	for _, secret := range []string{"", "  "} {
		if got := (&GithubProvider{secret: secret}).GetOptionalHeaderKeys(); len(got) != 0 {
			t.Errorf("GetOptionalHeaderKeys() with secret %q = %v, want none", secret, got)
		}
	}
}

func TestGithubProvider_GetProviderName(t *testing.T) {
	t.Parallel()
	if got := (&GithubProvider{}).GetProviderName(); got != GithubName {
		t.Errorf("GetProviderName() = %q, want %q", got, GithubName)
	}
}

func TestHashPayload(t *testing.T) {
	t.Parallel()
	// Well-known HMAC-SHA1 reference value.
	got := HashPayload("key", []byte("The quick brown fox jumps over the lazy dog"))
	want := "de7c9b85b8b78aa6bc8a7a36f70a90701c9db4d9"
	if got != want {
		t.Errorf("HashPayload() = %s, want %s", got, want)
	}
}

func TestHashPayloadSHA256(t *testing.T) {
	t.Parallel()
	// Well-known HMAC-SHA256 reference value.
	got := HashPayloadSHA256("key", []byte("The quick brown fox jumps over the lazy dog"))
	want := "f7bc83f430538424b13298e6aa6fb143ef4d59a14946175997479dbc2d1a3cd8"
	if got != want {
		t.Errorf("HashPayloadSHA256() = %s, want %s", got, want)
	}
}

func TestIsValidPayloadSHA256(t *testing.T) {
	t.Parallel()
	payload := []byte(`{"zen":"Keep it logically awesome."}`)
	hash := HashPayloadSHA256(githubTestSecret, payload)

	tests := []struct {
		name    string
		secret  string
		hash    string
		payload []byte
		want    bool
	}{
		{name: "matching hash", secret: githubTestSecret, hash: hash, payload: payload, want: true},
		{name: "wrong secret", secret: "other", hash: hash, payload: payload, want: false},
		{name: "tampered payload", secret: githubTestSecret, hash: hash, payload: []byte(string(payload) + " "), want: false},
		{name: "sha1 hash", secret: githubTestSecret, hash: HashPayload(githubTestSecret, payload), payload: payload, want: false},
		{name: "empty hash", secret: githubTestSecret, hash: "", payload: payload, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsValidPayloadSHA256(tt.secret, tt.hash, tt.payload); got != tt.want {
				t.Errorf("IsValidPayloadSHA256() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidPayload(t *testing.T) {
	t.Parallel()
	payload := []byte(`{"zen":"Keep it logically awesome."}`)
	hash := HashPayload(githubTestSecret, payload)

	tests := []struct {
		name    string
		secret  string
		hash    string
		payload []byte
		want    bool
	}{
		{name: "matching hash", secret: githubTestSecret, hash: hash, payload: payload, want: true},
		{name: "wrong secret", secret: "other", hash: hash, payload: payload, want: false},
		{name: "tampered payload", secret: githubTestSecret, hash: hash, payload: []byte(string(payload) + " "), want: false},
		{name: "empty hash", secret: githubTestSecret, hash: "", payload: payload, want: false},
		{name: "upper case hash", secret: githubTestSecret, hash: strings.ToUpper(hash), payload: payload, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsValidPayload(tt.secret, tt.hash, tt.payload); got != tt.want {
				t.Errorf("IsValidPayload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGithubProvider_Validate(t *testing.T) {
	t.Parallel()
	payload := []byte(`{"ref":"refs/heads/main","sender":{"login":"octocat"}}`)
	validSignature := SignaturePrefix + HashPayload(githubTestSecret, payload)
	validSignature256 := Signature256Prefix + HashPayloadSHA256(githubTestSecret, payload)
	invalidSignature256 := Signature256Prefix + HashPayloadSHA256("WrongSecret", payload)

	tests := []struct {
		name    string
		secret  string
		headers map[string]string
		payload []byte
		want    bool
	}{
		{
			name:    "valid signature",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature: validSignature},
			payload: payload,
			want:    true,
		},
		{
			name:    "signature made with another secret",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature: SignaturePrefix + HashPayload("WrongSecret", payload)},
			payload: payload,
			want:    false,
		},
		{
			name:    "proxy configured with another secret",
			secret:  "WrongSecret",
			headers: map[string]string{XHubSignature: validSignature},
			payload: payload,
			want:    false,
		},
		{
			name:    "tampered payload",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature: validSignature},
			payload: []byte(`{"ref":"refs/heads/main","sender":{"login":"mallory"}}`),
			want:    false,
		},
		{
			name:    "missing signature header",
			secret:  githubTestSecret,
			headers: map[string]string{},
			payload: payload,
			want:    false,
		},
		{
			name:    "nil headers",
			secret:  githubTestSecret,
			headers: nil,
			payload: payload,
			want:    false,
		},
		{
			name:    "empty signature",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature: ""},
			payload: payload,
			want:    false,
		},
		{
			name:    "signature without prefix",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature: HashPayload(githubTestSecret, payload)},
			payload: payload,
			want:    false,
		},
		{
			// Same length as a valid header, but with a different algorithm prefix.
			name:    "wrong prefix with correct length",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature: "sha2=" + HashPayload(githubTestSecret, payload)},
			payload: payload,
			want:    false,
		},
		{
			name:    "sha256 prefix",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature: "sha256=" + HashPayload(githubTestSecret, payload)},
			payload: payload,
			want:    false,
		},
		{
			name:    "truncated signature",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature: validSignature[:SignatureLength-1]},
			payload: payload,
			want:    false,
		},
		{
			name:    "signature with trailing data",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature: validSignature + "0"},
			payload: payload,
			want:    false,
		},
		{
			name:    "valid sha256 signature",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature256: validSignature256},
			payload: payload,
			want:    true,
		},
		{
			name:    "sha256 signature made with another secret",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature256: invalidSignature256},
			payload: payload,
			want:    false,
		},
		{
			name:    "sha256 signature over tampered payload",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature256: validSignature256},
			payload: []byte(`{"ref":"refs/heads/main","sender":{"login":"mallory"}}`),
			want:    false,
		},
		{
			name:    "sha256 signature without prefix",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature256: HashPayloadSHA256(githubTestSecret, payload)},
			payload: payload,
			want:    false,
		},
		{
			name:    "sha256 header carrying a sha1 signature",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature256: validSignature},
			payload: payload,
			want:    false,
		},
		{
			name:    "sha256 signature with wrong prefix and correct length",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature256: "sha512=" + HashPayloadSHA256(githubTestSecret, payload)},
			payload: payload,
			want:    false,
		},
		{
			name:    "truncated sha256 signature",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature256: validSignature256[:Signature256Length-1]},
			payload: payload,
			want:    false,
		},
		{
			name:    "valid sha256 takes precedence over invalid sha1",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature256: validSignature256, XHubSignature: SignaturePrefix + HashPayload("WrongSecret", payload)},
			payload: payload,
			want:    true,
		},
		{
			name:    "invalid sha256 is not rescued by valid sha1",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature256: invalidSignature256, XHubSignature: validSignature},
			payload: payload,
			want:    false,
		},
		{
			name:    "empty sha256 header is not rescued by valid sha1",
			secret:  githubTestSecret,
			headers: map[string]string{XHubSignature256: "", XHubSignature: validSignature},
			payload: payload,
			want:    false,
		},
		{
			name:    "signature in wrong header",
			secret:  githubTestSecret,
			headers: map[string]string{"X-Wrong-Signature": validSignature},
			payload: payload,
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &GithubProvider{secret: tt.secret}
			hook := Hook{Headers: tt.headers, Payload: tt.payload, RequestMethod: "POST"}
			if got := p.Validate(hook); got != tt.want {
				t.Errorf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGithubProvider_GetCommitter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		event   string
		payload string
		want    string
	}{
		{name: "push event", event: "push", payload: `{"sender":{"login":"pusher"}}`, want: "pusher"},
		{name: "pull request event", event: "pull_request", payload: `{"sender":{"login":"pr-author"}}`, want: "pr-author"},
		{
			name:    "issue comment event uses comment author",
			event:   "issue_comment",
			payload: `{"comment":{"user":{"login":"commenter"}},"sender":{"login":"sender"}}`,
			want:    "commenter",
		},
		{name: "push event with invalid json", event: "push", payload: `{`, want: ""},
		{name: "pull request event with invalid json", event: "pull_request", payload: `{`, want: ""},
		{name: "issue comment event with invalid json", event: "issue_comment", payload: `{`, want: ""},
		{name: "unsupported event", event: "ping", payload: `{"sender":{"login":"pinger"}}`, want: ""},
		{name: "missing event header", event: "", payload: `{"sender":{"login":"pusher"}}`, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hook := Hook{
				Headers: map[string]string{XGitHubEvent: tt.event},
				Payload: []byte(tt.payload),
			}
			if got := (&GithubProvider{}).GetCommitter(hook); got != tt.want {
				t.Errorf("GetCommitter() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEventName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		event Event
		want  string
	}{
		{GithubPushEvent, "push"},
		{GithubPullRequestEvent, "pull_request"},
		{GithubIssueCommentEvent, "issue_comment"},
		{Event("ping"), "unsupported"},
		{Event("push\nforged log line"), "unsupported"},
	}
	for _, tt := range tests {
		if got := eventName(tt.event); got != tt.want {
			t.Errorf("eventName(%q) = %q, want %q", tt.event, got, tt.want)
		}
	}
}
