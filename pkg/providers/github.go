package providers

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // G505: X-Hub-Signature is defined as HMAC-SHA1 and kept for senders without X-Hub-Signature-256
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

const (
	GithubPushEvent         Event = "push"
	GithubPullRequestEvent  Event = "pull_request"
	GithubIssueCommentEvent Event = "issue_comment"
)

// Header constants
const (
	XHubSignature    = "X-Hub-Signature"
	XHubSignature256 = "X-Hub-Signature-256"
	XGitHubEvent     = "X-GitHub-Event"
	XGitHubDelivery  = "X-GitHub-Delivery"
)

const (
	SignaturePrefix    = "sha1="
	SignatureLength    = 45
	Signature256Prefix = "sha256="
	Signature256Length = 71
	GithubName         = "github"
)

type GithubProvider struct {
	secret string
}

func NewGithubProvider(secret string) (*GithubProvider, error) {
	return &GithubProvider{
		secret: secret,
	}, nil
}

func (p *GithubProvider) GetHeaderKeys() []string {
	return []string{
		XGitHubDelivery,
		XGitHubEvent,
		ContentTypeHeader,
	}
}

// GetOptionalHeaderKeys returns the signature headers. They are copied into
// the hook when present; Validate decides whether the ones present suffice.
func (p *GithubProvider) GetOptionalHeaderKeys() []string {
	if len(strings.TrimSpace(p.secret)) > 0 {
		return []string{XHubSignature256, XHubSignature}
	}
	return nil
}

// Validate checks the webhook signature. X-Hub-Signature-256 (HMAC-SHA256) is
// used when present; X-Hub-Signature (HMAC-SHA1) only when it is absent.
func (p *GithubProvider) Validate(hook Hook) bool {
	if signature, ok := hook.Headers[XHubSignature256]; ok {
		if len(signature) != Signature256Length ||
			!strings.HasPrefix(signature, Signature256Prefix) {
			return false
		}
		return IsValidPayloadSHA256(p.secret, signature[len(Signature256Prefix):], hook.Payload)
	}

	githubSignature := hook.Headers[XHubSignature]
	if len(githubSignature) != SignatureLength ||
		!strings.HasPrefix(githubSignature, SignaturePrefix) {
		return false
	}

	return IsValidPayload(p.secret, githubSignature[len(SignaturePrefix):], hook.Payload)
}

func (p *GithubProvider) GetProviderName() string {
	return GithubName
}

func (p *GithubProvider) GetCommitter(hook Hook) string {
	eventType := Event(hook.Headers[XGitHubEvent])
	var pushPayloadData GithubPushPayload
	var pullRequestPayloadData GithubPullRequestPayload
	var issueCommentPayloadData GithubIssueCommentPayload

	log.Printf("Received event type: %s", eventName(eventType))
	switch eventType {
	case GithubPushEvent:
		if err := json.Unmarshal(hook.Payload, &pushPayloadData); err != nil {
			log.Printf("Github payload unmarshaling failed for Push event: %v", err)
			return ""
		}
		return pushPayloadData.Sender.Login
	case GithubPullRequestEvent:
		if err := json.Unmarshal(hook.Payload, &pullRequestPayloadData); err != nil {
			log.Printf("Github payload unmarshaling failed for Pull Request event: %v", err)
			return ""
		}
		return pullRequestPayloadData.Sender.Login
	case GithubIssueCommentEvent:
		if err := json.Unmarshal(hook.Payload, &issueCommentPayloadData); err != nil {
			log.Printf("Github payload unmarshaling failed for issue comment event: %v", err)
			return ""
		}
		return issueCommentPayloadData.Comment.User.Login
	}

	log.Printf("Event type is not supported")
	return ""
}

// eventName returns the name of a supported event, or "unsupported". The
// raw X-GitHub-Event header value is untrusted and is not logged.
func eventName(e Event) string {
	switch e {
	case GithubPushEvent:
		return "push"
	case GithubPullRequestEvent:
		return "pull_request"
	case GithubIssueCommentEvent:
		return "issue_comment"
	}
	return "unsupported"
}

// IsValidPayload checks if the github payload's hash fits with
// the hash computed by GitHub sent as a header
func IsValidPayload(secret, headerHash string, payload []byte) bool {
	hash := HashPayload(secret, payload)
	return hmac.Equal(
		[]byte(hash),
		[]byte(headerHash),
	)
}

// HashPayload computes the hash of payload's body according to the webhook's secret token
// see https://developer.github.com/webhooks/securing/#validating-payloads-from-github
// returning the hash as a hexadecimal string
func HashPayload(secret string, playloadBody []byte) string {
	hm := hmac.New(sha1.New, []byte(secret))
	hm.Write(playloadBody)
	sum := hm.Sum(nil)
	return fmt.Sprintf("%x", sum)
}

// IsValidPayloadSHA256 reports whether headerHash is the hex encoded
// HMAC-SHA256 of payload under secret.
func IsValidPayloadSHA256(secret, headerHash string, payload []byte) bool {
	return hmac.Equal([]byte(HashPayloadSHA256(secret, payload)), []byte(headerHash))
}

// HashPayloadSHA256 returns the hex encoded HMAC-SHA256 of payload under
// secret, as sent by GitHub in X-Hub-Signature-256.
func HashPayloadSHA256(secret string, payload []byte) string {
	hm := hmac.New(sha256.New, []byte(secret))
	hm.Write(payload)
	return hex.EncodeToString(hm.Sum(nil))
}
