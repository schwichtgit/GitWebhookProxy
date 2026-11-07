package providers

import (
	"errors"
	"strings"
)

const (
	GithubProviderKind            = "github"
	GitlabProviderKind            = "gitlab"
	ContentTypeHeader             = "Content-Type"
	DefaultContentTypeHeaderValue = "application/json"
)

// Event defines a provider hook event type
type Event string

type Provider interface {
	GetHeaderKeys() []string
	Validate(hook Hook) bool
	GetCommitter(hook Hook) string
	GetProviderName() string
}

// Compile-time checks that the provider types implement Provider.
var (
	_ Provider = (*GithubProvider)(nil)
	_ Provider = (*GitlabProvider)(nil)

	_ OptionalHeaderProvider = (*GithubProvider)(nil)
)

func NewProvider(provider string, secret string) (Provider, error) {
	if len(provider) == 0 {
		return nil, errors.New("empty provider string specified")
	}

	switch strings.ToLower(provider) {
	case GithubProviderKind:
		return NewGithubProvider(secret)
	case GitlabProviderKind:
		return NewGitlabProvider(secret)
	default:
		return nil, errors.New("unknown git provider '" + provider + "' specified")
	}
}

// OptionalHeaderProvider is implemented by providers that accept headers
// which are copied into the hook only when the request carries them.
type OptionalHeaderProvider interface {
	GetOptionalHeaderKeys() []string
}

type Hook struct {
	Payload       []byte
	Headers       map[string]string
	RequestMethod string
}
