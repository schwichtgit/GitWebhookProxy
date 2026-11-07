# Build and release

Everything goes through the top-level `Makefile`. Run `make` (or `make help`)
to list targets and the current values of the main variables.

## Requirements

- Go, version from `go.mod`
- golangci-lint v2, govulncheck
- Docker with buildx, or Podman
- trivy, helm (for `image-scan` and `helm-lint`)

## Versioning

The version is taken from git: `git describe --tags --always --dirty`. Tag a
commit `vX.Y.Z` to release it. There is no version file. The value is linked
into the binary with `-X main.version=...` and printed at startup.

## Make targets

| Target       | What it does                                                                                                 |
| ------------ | ------------------------------------------------------------------------------------------------------------ |
| `build`      | Builds `./gitwebhookproxy` with `CGO_ENABLED=0` and `-trimpath`.                                             |
| `test`       | `go test -race ./...`.                                                                                       |
| `cover`      | Same as `test` with a coverage profile in `coverage.out`. Fails if total coverage is below `COVER_MIN` (85). |
| `lint`       | `golangci-lint run` with `.golangci.yml`.                                                                    |
| `vuln`       | `govulncheck ./...`.                                                                                         |
| `fmt`        | `go fmt ./...`.                                                                                              |
| `tidy-check` | Fails if `go mod tidy` would change `go.mod` or `go.sum`.                                                    |
| `image`      | Builds the container image. Alias: `docker`.                                                                 |
| `image-scan` | `trivy image`, fails on fixable HIGH or CRITICAL findings.                                                   |
| `helm-lint`  | `helm lint --strict` and `helm template` of the chart, including each `ci/*-values.yaml`.                    |
| `clean`      | Removes the binary, `coverage.out` and `dist/`.                                                              |

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
- The runtime image is `gcr.io/distroless/static-debian12:nonroot`. It has CA
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

`.github/workflows/ci.yml` runs on pull requests and pushes to `master`:

- `go mod tidy` check, `go vet`, golangci-lint, `make cover`, govulncheck
- `make helm-lint`, kubeconform (strict, Kubernetes 1.30 and 1.33 schemas,
  missing CRD schemas such as GKE BackendConfig are skipped), and
  helm-unittest when the chart has tests
- hadolint, image build without push, trivy scan of the image

## Release

Push a tag `vX.Y.Z`. `.github/workflows/release.yml` then:

1. Runs GoReleaser (`.goreleaser.yaml`): binaries for linux, darwin and
   windows on amd64 and arm64, archives, checksums and changelog, published
   as a GitHub release.
2. Builds and pushes the multi-arch image (linux/amd64, linux/arm64) to
   `ghcr.io/schwichtgit/gitwebhookproxy`, tagged `X.Y.Z` and `X.Y`, with
   provenance and SBOM attestations, and signs it with cosign (keyless).
3. Packages the Helm chart with version and appVersion `X.Y.Z` and pushes it
   to `oci://ghcr.io/schwichtgit/charts`.

Verify an image signature:

```bash
cosign verify ghcr.io/schwichtgit/gitwebhookproxy:X.Y.Z \
  --certificate-identity-regexp '^https://github.com/schwichtgit/GitWebhookProxy/' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

Install the chart:

```bash
helm install gitwebhookproxy oci://ghcr.io/schwichtgit/charts/gitwebhookproxy \
  --version X.Y.Z --set config.upstreamURL=https://jenkins.example.com
```

## Dependency updates

Dependabot (`.github/dependabot.yml`) checks Go modules, the Dockerfile base
images and GitHub Actions weekly, grouping minor and patch updates.
