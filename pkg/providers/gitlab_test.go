package providers

import (
	"reflect"
	"strings"
	"testing"
)

const (
	gitlabTestSecret = "myGitLabTestSecret"
)

func TestNewGitlabProvider(t *testing.T) {
	type args struct {
		secret string
	}
	tests := []struct {
		name    string
		args    args
		want    *GitlabProvider
		wantErr bool
	}{
		{
			name: "TestNewGitlabProviderWithCorrectSecret",
			args: args{
				secret: gitlabTestSecret,
			},
			want: &GitlabProvider{
				secret: gitlabTestSecret,
			},
			wantErr: false,
		},
		{
			name: "TestNewGitlabProviderWithEmptySecret",
			args: args{
				secret: "",
			},
			want: &GitlabProvider{
				secret: "",
			},
			wantErr: false,
		},
		{
			name:    "TestNewGitlabProviderWithNoSecret",
			args:    args{},
			want:    &GitlabProvider{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGitlabProvider(tt.args.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGitlabProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewGitlabProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGitlabProvider_GetHeaderKeys(t *testing.T) {
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
			want: []string{XGitlabEvent, ContentTypeHeader},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &GitlabProvider{
				secret: tt.fields.secret,
			}
			if got := p.GetHeaderKeys(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GitlabProvider.GetHeaderKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGitlabProvider_Validate(t *testing.T) {
	type fields struct {
		secret string
	}
	type args struct {
		hook Hook
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "TestValidateWithEmptyTokenValue",
			fields: fields{
				secret: gitlabTestSecret,
			},
			args: args{
				hook: Hook{
					Headers: map[string]string{
						XGitlabToken: "",
					},
				},
			},
			want: false,
		},
		{
			name: "TestValidateWithEmptyHeaders",
			fields: fields{
				secret: gitlabTestSecret,
			},
			args: args{
				hook: Hook{
					Headers: map[string]string{},
				},
			},
			want: false,
		},
		{
			name: "TestValidateWithWrongTokenValue",
			fields: fields{
				secret: gitlabTestSecret,
			},
			args: args{
				hook: Hook{
					Headers: map[string]string{
						XGitlabToken: "IncorrectSecret",
					},
					Payload: nil,
				},
			},
			want: false,
		},
		{
			name: "TestValidateWithCorrectTokenValue",
			fields: fields{
				secret: gitlabTestSecret,
			},
			args: args{
				hook: Hook{
					Headers: map[string]string{
						XGitlabToken: gitlabTestSecret,
					},
					Payload: nil,
				},
			},
			want: true,
		},
		{
			name: "TestValidateWithWrongHeaderKey",
			fields: fields{
				secret: gitlabTestSecret,
			},
			args: args{
				hook: Hook{
					Headers: map[string]string{
						"X-Wrong-Token": "IncorrectSecret",
					},
					Payload: nil,
				},
			},
			want: false,
		},
		{
			name: "TestValidateWithNilHeaders",
			fields: fields{
				secret: gitlabTestSecret,
			},
			args: args{
				hook: Hook{
					Headers: nil,
					Payload: nil,
				},
			},
			want: false,
		},
		{
			name: "TestValidateWithNoHookArg",
			fields: fields{
				secret: gitlabTestSecret,
			},
			args: args{},
			want: false,
		},
		{
			name: "TestValidateWithWrongSecretInProxy",
			fields: fields{
				secret: "WrongSecret",
			},
			args: args{
				hook: Hook{
					Headers: map[string]string{
						XGitlabToken: gitlabTestSecret,
					},
					Payload: nil,
				},
			},
			want: false,
		},
		{
			name: "TestValidateWithEmptySecretInProxy",
			fields: fields{
				secret: "",
			},
			args: args{
				hook: Hook{
					Headers: map[string]string{
						XGitlabToken: gitlabTestSecret,
					},
					Payload: nil,
				},
			},
			want: false,
		},
		{
			name:   "TestValidateWithNoSecretInProxy",
			fields: fields{},
			args: args{
				hook: Hook{
					Headers: map[string]string{
						XGitlabToken: gitlabTestSecret,
					},
					Payload: nil,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &GitlabProvider{
				secret: tt.fields.secret,
			}
			if got := p.Validate(tt.args.hook); got != tt.want {
				t.Errorf("GitlabProvider.Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGitlabProvider_ValidateTrimsWhitespace(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		secret string
		token  string
		want   bool
	}{
		{name: "token with surrounding whitespace", secret: gitlabTestSecret, token: "  " + gitlabTestSecret + "\n", want: true},
		{name: "secret with surrounding whitespace", secret: " " + gitlabTestSecret + " ", token: gitlabTestSecret, want: true},
		{name: "token is a prefix of the secret", secret: gitlabTestSecret, token: gitlabTestSecret[:len(gitlabTestSecret)-1], want: false},
		{name: "secret is a prefix of the token", secret: gitlabTestSecret, token: gitlabTestSecret + "x", want: false},
		{name: "case differs", secret: gitlabTestSecret, token: strings.ToUpper(gitlabTestSecret), want: false},
		{name: "whitespace only token", secret: gitlabTestSecret, token: "   ", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &GitlabProvider{secret: tt.secret}
			hook := Hook{Headers: map[string]string{XGitlabToken: tt.token}}
			if got := p.Validate(hook); got != tt.want {
				t.Errorf("Validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGitlabProvider_GetHeaderKeysWithSecret(t *testing.T) {
	t.Parallel()
	p := &GitlabProvider{secret: gitlabTestSecret}
	want := []string{XGitlabEvent, XGitlabToken, ContentTypeHeader}
	if got := p.GetHeaderKeys(); !reflect.DeepEqual(got, want) {
		t.Errorf("GetHeaderKeys() = %v, want %v", got, want)
	}
}

func TestGitlabProvider_GetProviderName(t *testing.T) {
	t.Parallel()
	if got := (&GitlabProvider{}).GetProviderName(); got != GitlabName {
		t.Errorf("GetProviderName() = %q, want %q", got, GitlabName)
	}
}

func TestGitlabProvider_GetCommitter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		event   string
		payload string
		want    string
	}{
		{name: "push hook", event: string(GitlabPushEvent), payload: `{"user_username":"jsmith"}`, want: "jsmith"},
		{name: "merge request hook is not supported", event: string(GitlabMergeRequestEvent), payload: `{"user_username":"jsmith"}`, want: ""},
		{name: "unknown event", event: "Tag Push Hook", payload: `{"user_username":"jsmith"}`, want: ""},
		{name: "invalid json", event: string(GitlabPushEvent), payload: `not json`, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hook := Hook{
				Headers: map[string]string{XGitlabEvent: tt.event},
				Payload: []byte(tt.payload),
			}
			if got := (&GitlabProvider{}).GetCommitter(hook); got != tt.want {
				t.Errorf("GetCommitter() = %q, want %q", got, tt.want)
			}
		})
	}
}
