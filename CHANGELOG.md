# Changelog

All notable changes to this fork are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions follow
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed

- Go 1.27.4 (go.mod, mise toolchain and the `golang:1.27.4-alpine3.24`
  builder image, pinned by digest). govulncheck reports no known
  vulnerabilities.

## [0.3.1] - 2026-11-08

### Fixed

- Release: the Helm chart signature could not be pushed because cosign had
  no registry credentials (`helm registry login` does not write the Docker
  config). Chart 0.3.0 was published without a signature; use 0.3.1.

## [0.3.0] - 2026-11-08

First release of the fork `schwichtgit/GitWebhookProxy`.

### Upgrade notes

Read these before upgrading from upstream 0.2.x:

- **The Helm chart cannot be upgraded in place.** Chart 0.3.0 changes the
  Deployment selector labels, which Kubernetes does not allow to change, so
  `helm upgrade` from 0.2.x fails. Uninstall and reinstall the release, or
  install 0.3.0 under a new release name and switch traffic. Webhook
  deliveries fail while no proxy is running; redeliver them afterwards. Steps
  are in [RUNBOOK.md](RUNBOOK.md#deploy-and-upgrade).
- **The allowed-users filter now takes effect.** Chart 0.2.x passed
  `allowedUsers` as `GWP_ALLOWEDUSERS`, which the binary ignores, so the filter
  was never applied. With 0.3.0 it is passed as `GWP_ALLOWEDUSER`. If
  `allowedUsers` is set in your values, webhooks from all other senders are
  dropped from now on. Check the list before upgrading.
- **Chart values were renamed and restructured** (`gitWebhookProxy.*` is gone).
  See the migration table in the
  [chart README](deployments/kubernetes/chart/gitwebhookproxy/README.md#upgrading-from-chart-02x).
- **New image location and format.** Images are published to
  `ghcr.io/schwichtgit/gitwebhookproxy` with tags `X.Y.Z` and `X.Y` (no `v`
  prefix, no `latest`). The image is distroless, runs as uid 65532, has no
  shell, and its entrypoint is `/gitwebhookproxy` (was `/bin/GitWebhookProxy`).
- **Signature checks run before user filtering.** Requests with an invalid or
  missing signature now get `400 Error validating Hook` even if the sender is
  in `ignoredUsers`. Previously they got `200 Ignoring request for user`.
- **Allowed paths match at path segment boundaries.** `/path1` allows `/path1`
  and `/path1/...` but no longer `/path10`.
- **Error responses changed.** They no longer contain the upstream URL. An
  unreachable or unusable upstream returns `502` instead of `500`.
- **Go module path** is `github.com/schwichtgit/GitWebhookProxy`.

### Added

- GitHub `X-Hub-Signature-256` (HMAC-SHA256) validation. It is used when
  present; `X-Hub-Signature` (HMAC-SHA1) is used only when it is absent.
- FIPS 140-3 mode by default: binaries and images are built with
  `GOFIPS140=v1.0.0` (CMVP certificate #5247). The mode and module version are
  logged at startup.
- `--insecureSkipVerify` / `GWP_INSECURESKIPVERIFY` to accept self-signed
  upstream certificates (off by default).
- Version reported at startup, injected from the git tag.
- Release artifacts signed keyless with cosign: multi-arch image index and
  platform manifests, Helm chart, and the checksums file covering the
  archives. Per-platform SPDX SBOMs (syft) attached as signed attestations,
  plus SLSA provenance.
- Helm chart 0.3.0: values schema, restricted Pod Security defaults,
  ServiceAccount, HPA, PDB, NetworkPolicy, GKE BackendConfig/FrontendConfig,
  `helm test`, helm-unittest suites, and an optional Kyverno policy that
  verifies the image signature and SBOM at admission.
- Pinned toolchain in `mise.toml` / `mise.lock` used locally and in CI.
- `make qa` static QA gates, run in CI: gofmt, go vet, go mod tidy,
  golangci-lint, govulncheck, markdownlint-cli2, prettier, actionlint,
  hadolint, helm lint, helm-unittest, kubeconform, Kyverno Pod Security, and
  trivy secret and misconfiguration scans.
- Trivy CVE and secret scan of the container image in CI and as a release
  gate before signing.
- GitHub Actions CI (QA gates, tests with an 85% coverage gate, multi-arch
  image build and scan) and a tag-driven release workflow.
- Dependabot for Go modules, base images and GitHub Actions; a weekly
  workflow bumps the mise tool pins.
- [RUNBOOK.md](RUNBOOK.md).

### Changed

- Go 1.27.3; go-chi/chi v5.5.0, spf13/pflag v1.1.0.
- Container image built from a single multi-stage Dockerfile, cross-compiled
  for linux/amd64 and linux/arm64, on `gcr.io/distroless/static-debian12:nonroot`.
- The HTTP server sets read, write and idle timeouts.
- GitLab token comparison is constant-time.
- Request-derived values in log lines are quoted. Responses that echo payload
  data are `text/plain` with `X-Content-Type-Options: nosniff`.
- A missing `--upstreamURL` prints the usage and exits with status 2 instead
  of a panic trace.

### Fixed

- Upstream TLS certificates are verified (previously verification was
  disabled for all upstreams).
- The upstream response body was not closed for error responses, leaking a
  connection per failed request.
- The 0.2.x chart did not render (`config.rateLimit`, `config.retry` and
  Ingress keys were referenced but not defined) and passed allowed users under
  the wrong variable name.
- The computed HMAC of each payload is no longer logged.
- CVE fixes: GO-2026-6809 (crypto/x509, Go 1.27.3), GO-2025-3770 (chi open
  redirect), CVE-2026-4738 (zlib in the old Alpine runtime image).

### Removed

- Legacy build path (`Dockerfile.build` with packr, `Dockerfile.run`), the
  upstream Jenkins pipeline (`Jenkinsfile`, `stk.yaml`), `renovate.json` and
  `.VERSION`.
- Raw Kubernetes manifests under `deployments/kubernetes`; the Helm chart is
  the only supported deployment method.
- `examples/` (a copy of `main`).
- 32-bit and ARMv7 release binaries.

## 0.2.x

Releases up to v0.2.79 were published by
[stakater/GitWebhookProxy](https://github.com/stakater/GitWebhookProxy/releases).

[0.3.1]: https://github.com/schwichtgit/GitWebhookProxy/releases/tag/v0.3.1
[0.3.0]: https://github.com/schwichtgit/GitWebhookProxy/releases/tag/v0.3.0
