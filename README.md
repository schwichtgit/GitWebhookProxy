# GitWebhookProxy

GitWebhookProxy receives webhooks from GitHub or GitLab, checks their signature
or token, filters them by path and user, and forwards the accepted ones to an
upstream service such as Jenkins. It lets you expose one small, hardened
endpoint to the internet instead of the CI server itself.

This repository is a fork of
[stakater/GitWebhookProxy](https://github.com/stakater/GitWebhookProxy) and is
maintained independently.

- Go 1.27, single static binary, FIPS 140-3 mode on by default
- GitHub `X-Hub-Signature-256` (HMAC-SHA256) with fallback to `X-Hub-Signature`
  (HMAC-SHA1); GitLab `X-Gitlab-Token`
- Container image: distroless, non-root, about 10 MB, signed with cosign
  (keyless) and shipped with a signed SPDX SBOM
- Helm chart with restricted Pod Security defaults and optional Kyverno image
  verification

Operating the service: [RUNBOOK.md](RUNBOOK.md). Changes and upgrade notes:
[CHANGELOG.md](CHANGELOG.md).

## How a request is handled

For every `POST`, in this order:

1. The path must match `allowedPaths` (if set), otherwise `403`.
2. The provider headers must be present, otherwise `400`.
3. If a secret is configured, the signature (GitHub) or token (GitLab) must be
   valid, otherwise `400 Error validating Hook`.
4. The sender is checked against `ignoredUsers` and `allowedUser`. Filtered
   requests get `200 Ignoring request for user: <name>` and are not forwarded.
   GitHub events without a sender are always ignored.
5. The request is forwarded to `upstreamURL` + the request path and query, with
   the body and the provider headers only. The upstream status and body are
   returned. Upstream status codes of 400 and above are passed through; an
   unreachable upstream gives `502`.

`GET /health` returns `200`.

## Configuration

Every flag can also be set as an environment variable with the `GWP_` prefix.
Flags win over environment variables.

| Flag | Environment variable | Default | Description |
| --- | --- | --- | --- |
| `--upstreamURL` | `GWP_UPSTREAMURL` | (required) | Base URL that accepted webhooks are forwarded to. |
| `--provider` | `GWP_PROVIDER` | `github` | `github` or `gitlab`. |
| `--secret` | `GWP_SECRET` | `""` | Webhook secret (GitHub) or token (GitLab). If empty, requests are not validated. |
| `--allowedPaths` | `GWP_ALLOWEDPATHS` | `""` (all) | Comma-separated paths. `/a` allows `/a` and `/a/...`, not `/ab`. |
| `--ignoredUsers` | `GWP_IGNOREDUSERS` | `""` | Comma-separated senders to drop. |
| `--allowedUser` | `GWP_ALLOWEDUSER` | `""` (all) | Comma-separated senders to accept; everyone else is dropped. |
| `--listen` | `GWP_LISTEN` | `:8080` | Listen address. |
| `--insecureSkipVerify` | `GWP_INSECURESKIPVERIFY` | `false` | Skip TLS verification of the upstream. Only for trusted upstreams with self-signed certificates. |

Run the proxy without a secret only behind another layer that authenticates
the sender.

## Run

### Kubernetes (Helm)

```sh
kubectl create secret generic gitwebhookproxy-secret \
  --from-literal=secret='<webhook secret>'

helm install gwp oci://ghcr.io/schwichtgit/charts/gitwebhookproxy \
  --version 0.3.0 \
  --set config.upstreamURL=http://jenkins.jenkins.svc:8080 \
  --set secret.existingSecret=gitwebhookproxy-secret

helm test gwp
```

Requires Kubernetes 1.27 or later. See the
[chart README](deployments/kubernetes/chart/gitwebhookproxy/README.md) for all
values, Ingress and GKE examples, Kyverno image verification, and the
migration from chart 0.2.x (a reinstall is required).

### Container

```sh
docker run --rm -p 8080:8080 \
  -e GWP_UPSTREAMURL=http://jenkins.example.com \
  -e GWP_SECRET='<webhook secret>' \
  ghcr.io/schwichtgit/gitwebhookproxy:0.3.0
```

Images are tagged `X.Y.Z` and `X.Y` for `linux/amd64` and `linux/arm64`. There
is no `latest` tag. The image has no shell; the entrypoint is
`/gitwebhookproxy` and flags are passed as container arguments.

### Binary

Release archives for Linux, macOS and Windows (amd64, arm64) are attached to
each [GitHub release](https://github.com/schwichtgit/GitWebhookProxy/releases).

```sh
./gitwebhookproxy --upstreamURL=http://jenkins.example.com --secret='<webhook secret>'
```

## Configure the webhook

- GitHub: Payload URL `https://<proxy host>/<path>`, content type
  `application/json`, and the same secret as the proxy. GitHub sends both
  signature headers; the proxy uses `X-Hub-Signature-256`. Step by step:
  [docs/github-webhook.md](docs/github-webhook.md).
- GitLab: URL `https://<proxy host>/<path>` and the proxy secret as the
  secret token.

## Verify a release

Everything is signed keyless by the release workflow. The certificate identity
is the workflow file on the release tag:

```sh
ID='https://github.com/schwichtgit/GitWebhookProxy/.github/workflows/release.yml@refs/tags/v0.3.0'
ISSUER='https://token.actions.githubusercontent.com'

# Image signature
cosign verify --certificate-identity "$ID" --certificate-oidc-issuer "$ISSUER" \
  ghcr.io/schwichtgit/gitwebhookproxy:0.3.0

# SBOM attestation of one platform image
cosign verify-attestation --type spdxjson \
  --certificate-identity "$ID" --certificate-oidc-issuer "$ISSUER" \
  ghcr.io/schwichtgit/gitwebhookproxy@<linux/amd64 digest>

# Release archives: verify the checksums file, then the archives against it
cosign verify-blob --bundle gitwebhookproxy_0.3.0_checksums.txt.sigstore.json \
  --certificate-identity "$ID" --certificate-oidc-issuer "$ISSUER" \
  gitwebhookproxy_0.3.0_checksums.txt
sha256sum --ignore-missing -c gitwebhookproxy_0.3.0_checksums.txt

# Helm chart
cosign verify --certificate-identity "$ID" --certificate-oidc-issuer "$ISSUER" \
  ghcr.io/schwichtgit/charts/gitwebhookproxy:0.3.0
```

## FIPS 140-3

Release binaries, the container image and `make build` are built with
`GOFIPS140=v1.0.0`, the CMVP-validated Go Cryptographic Module
(certificate #5247). The binary runs in FIPS 140-3 mode without further settings and logs
`FIPS 140-3 mode enabled (Go Cryptographic Module v1.0.0)` at startup. In this
mode `crypto/tls` negotiates only approved protocol versions, cipher suites and
key exchanges. Build with `GOFIPS140=off` to turn it off. See
<https://go.dev/doc/security/fips140>.

## Development

```sh
make help        # list targets
make build       # ./gitwebhookproxy
make cover       # tests with race detector, fails below 85% coverage
make lint vuln   # golangci-lint, govulncheck
make image       # container image
make helm-lint   # lint and render the chart
```

Build system, CI and the release process: [build/README.md](build/README.md).
Releases are cut by pushing a `vX.Y.Z` tag; dependencies are kept current by
Dependabot.

## License

Apache License 2.0, see [LICENSE](LICENSE).
