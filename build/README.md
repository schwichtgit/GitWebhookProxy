# Build and release

Everything goes through the top-level `Makefile`. Run `make` (or `make help`)
to list targets and the current values of the main variables.

## Toolchain

All tools are pinned in `mise.toml`, with checksums in `mise.lock`: Go,
Node, golangci-lint, govulncheck, markdownlint-cli2, prettier, actionlint,
shellcheck, hadolint, helm, kubectl, kubeconform, kyverno, goreleaser,
cosign, syft and trivy. CI installs exactly these versions with
`jdx/mise-action`, and the mise version itself is pinned in the workflows.

```bash
mise install          # or: make tools
mise exec -- make qa  # if mise is not activated in your shell
```

Docker with buildx, or Podman, is needed for `make image`.

The helm-unittest plugin and the kyverno/policies checkout are not in the
mise registry; their versions are pinned in the Makefile
(`HELM_UNITTEST_VERSION`, `KYVERNO_POLICIES_REF`).

## Versioning

The version is taken from git: `git describe --tags --always --dirty`. Tag a
commit `vX.Y.Z` to release it. There is no version file. The value is linked
into the binary with `-X main.version=...` and printed at startup.

## Make targets

| Target          | What it does                                                                                    |
| --------------- | ----------------------------------------------------------------------------------------------- |
| `tools`         | `mise install`.                                                                                 |
| `qa`            | All static QA gates below, in one run. CI runs exactly this.                                    |
| `build`         | Builds `./gitwebhookproxy` with `CGO_ENABLED=0` and `-trimpath`.                                |
| `test`          | `go test -race ./...`.                                                                          |
| `cover`         | `test` with a coverage profile in `coverage.out`; fails below `COVER_MIN` (85%).                |
| `fmt`           | `go fmt ./...`.                                                                                 |
| `fmt-check`     | Fails if `gofmt -l` lists files.                                                                |
| `vet`           | `go vet ./...`.                                                                                 |
| `tidy-check`    | Fails if `go mod tidy` would change `go.mod` or `go.sum`.                                       |
| `lint`          | golangci-lint with `.golangci.yml`.                                                             |
| `vuln`          | govulncheck.                                                                                    |
| `docs-lint`     | markdownlint-cli2 (`.markdownlint-cli2.yaml`) and `prettier --check` on all markdown.           |
| `actionlint`    | actionlint (with shellcheck) on the workflows.                                                  |
| `hadolint`      | hadolint on the Dockerfile.                                                                     |
| `helm-lint`     | `helm lint --strict` and `helm template`, including each `ci/*-values.yaml`.                    |
| `helm-unittest` | Chart unit tests in `tests/`.                                                                   |
| `kubeconform`   | Rendered manifests against Kubernetes 1.30 and 1.33 schemas (CRDs without schemas are skipped). |
| `kyverno-check` | Rendered manifests against the Kyverno Pod Security baseline and restricted policies.           |
| `secret-scan`   | `trivy fs --scanners secret` on the repository.                                                 |
| `config-scan`   | `trivy config` on the Dockerfile, chart and workflows; fails on HIGH or CRITICAL.               |
| `image`         | Builds the container image. Alias: `docker`.                                                    |
| `image-scan`    | `trivy image`; fails on fixable HIGH/CRITICAL CVEs or on a secret of any severity.              |
| `clean`         | Removes the binary, `coverage.out`, `dist/` and `.cache/`.                                      |

Variables can be overridden on the command line, for example
`make image IMAGE=registry.example.com/gwp VERSION=v1.2.3`.

| Variable        | Default                                      |
| --------------- | -------------------------------------------- |
| `VERSION`       | `git describe --tags --always --dirty`       |
| `IMAGE`         | `ghcr.io/schwichtgit/gitwebhookproxy`        |
| `IMAGE_TAG`     | `VERSION` with `+` replaced by `_`           |
| `PLATFORMS`     | `linux/<host arch>`                          |
| `PUSH`          | `0`; set to `1` to push the image            |
| `CONTAINER_CLI` | `docker` if buildx works, otherwise `podman` |
| `GOFIPS140`     | `v1.0.0`                                     |
| `GOPROXY`       | empty (Go default)                           |
| `COVER_MIN`     | `85`                                         |

## Container image

`build/package/Dockerfile` is the only Dockerfile.

- The builder stage runs on the build host (`--platform=$BUILDPLATFORM`) and
  cross-compiles for `TARGETOS`/`TARGETARCH`, so multi-arch builds do not need
  emulation for the Go compile.
- Module and build caches use BuildKit cache mounts.
- The runtime image is `gcr.io/distroless/static-debian13:nonroot`. It has CA
  certificates, no shell, and runs as uid/gid 65532.
- The binary is `/gitwebhookproxy` and is the entrypoint. Pass flags as
  container arguments or use the `GWP_*` environment variables.
- The container listens on port 8080; the health endpoint is `/health`.
- Both base images are pinned by digest. Dependabot updates the pins.

Build arguments: `VERSION`, `REVISION` (OCI labels), `GOFIPS140`, `GOPROXY`.

Local image for the host architecture:

```bash
make image
make image-scan
docker run --rm -p 8080:8080 ghcr.io/schwichtgit/gitwebhookproxy:$(git describe --tags --always --dirty) \
  --upstreamURL=https://jenkins.example.com
```

Multi-arch build and push (docker buildx cannot load a multi-platform image
into the local store, so `PUSH=1` is needed):

```bash
make image PLATFORMS=linux/amd64,linux/arm64 PUSH=1
```

## FIPS 140-3

`GOFIPS140=v1.0.0` is set for `make build`, `make test`, the Dockerfile and
GoReleaser. The binary is built against the CMVP-validated Go Cryptographic
Module (certificate #5247) and runs in FIPS 140-3 mode by default. At startup
it logs:

```text
FIPS 140-3 mode enabled (Go Cryptographic Module v1.0.0)
```

To build without it, set `GOFIPS140=off` (`make build GOFIPS140=off`, or
`--build-arg GOFIPS140=off`). Release binaries are built for amd64 and arm64
only; FIPS mode is not supported on 32-bit Windows.

See <https://go.dev/doc/security/fips140>.

## CI

`.github/workflows/ci.yml` runs on pull requests and pushes to `master`, on
`ubuntu-24.04` runners, with the tools from `mise.toml`:

- `qa`: `make qa` (all static gates, including trivy secret and
  misconfiguration scans)
- `test`: `make cover`
- `image`: multi-arch build (linux/amd64, linux/arm64) without push, then
  `make image-scan` on the host-architecture image (trivy CVE and secret
  scan)

Pull requests that only change documentation (markdown files, `docs/`,
`LICENSE`, issue templates) run `make docs-lint` in the `qa` job and skip
`test` and `image`. The jobs are required checks in the `master` ruleset, so
the workflow always starts; a skipped job counts as passed. Pushes to
`master` always run everything.

## Scheduled security scans

`.github/workflows/security.yml` runs every Monday, on demand
(`workflow_dispatch`), and on pull requests that change it:

- trivy scan of the latest released image, per platform (CVEs and secrets)
- govulncheck on the default branch and on the latest release tag
- trivy secret and misconfiguration scan of the repository

Results are uploaded to the Security tab as SARIF. Fixable HIGH/CRITICAL CVEs
or secrets in the released image, and reachable Go vulnerabilities, fail the
job. A failure means a new advisory affects the current release: fix it and
cut a patch release (see RUNBOOK.md, Vulnerability response).

## Release

Push a tag `vX.Y.Z`. `.github/workflows/release.yml` then:

1. Runs GoReleaser (`.goreleaser.yaml`): binaries for linux, darwin and
   windows on amd64 and arm64, archives, checksums and changelog, published
   as a GitHub release.
   GoReleaser also writes an SPDX SBOM per archive (syft) and signs the
   checksums file with cosign (keyless).
2. Builds and pushes the multi-arch image (linux/amd64, linux/arm64) to
   `ghcr.io/schwichtgit/gitwebhookproxy`, tagged `X.Y.Z` and `X.Y`, with SLSA
   provenance. Each platform image is scanned with trivy (CVEs and secrets)
   before anything is signed. The index and platform manifests are then
   signed with cosign (keyless), and a syft SPDX SBOM is attached to each
   platform manifest as a signed attestation.
3. Packages the Helm chart with version and appVersion `X.Y.Z`, pushes it to
   `oci://ghcr.io/schwichtgit/charts` and signs it with cosign (keyless).

Every job verifies its signatures against the workflow identity before it
finishes.

Verify an image signature:

```bash
cosign verify ghcr.io/schwichtgit/gitwebhookproxy:X.Y.Z \
  --certificate-identity https://github.com/schwichtgit/GitWebhookProxy/.github/workflows/release.yml@refs/tags/vX.Y.Z \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

Install the chart:

```bash
helm install gitwebhookproxy oci://ghcr.io/schwichtgit/charts/gitwebhookproxy \
  --version X.Y.Z --set config.upstreamURL=https://jenkins.example.com
```

## Dependency updates

- Dependabot (`.github/dependabot.yml`) checks Go modules, the Dockerfile
  base images and GitHub Actions every Saturday. A cooldown holds back new
  versions (major 14 days, minor 7, patch 3, other 7) so a release that is
  pulled shortly after publication never reaches a pull request. Security
  updates are not subject to the cooldown.
- Dependabot cannot read `mise.toml`. `.github/workflows/mise-update.yml`
  runs `mise upgrade --bump --minimum-release-age 7d` and `mise lock` every
  Monday and opens a pull request with the changes. Versions published in the
  last 7 days are skipped, matching the Dependabot cooldown. Node stays on its
  LTS major (`NODE_MAJOR`). Pull requests opened with `GITHUB_TOKEN` do not
  start CI on their own; close and reopen the PR to run it.
- `MISE_VERSION` in the workflows and the two Makefile pins above are updated
  by hand.
