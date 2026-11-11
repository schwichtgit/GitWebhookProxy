# syntax=docker/dockerfile:1

# The builder runs on the build host and cross-compiles for the target platform.
# hadolint ignore=DL3029
FROM --platform=$BUILDPLATFORM golang:1.27.4-alpine3.24@sha256:0a5f7fe796599ff487605987432e7619ae45beeb2fba4c2ce0b907879932aebf AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
# Build against the CMVP-validated Go Cryptographic Module (certificate #5247).
# This also turns on FIPS 140-3 mode at runtime. See https://go.dev/doc/security/fips140
ARG GOFIPS140=v1.0.0
# Optional module proxy override; unset or empty uses the Go default.
ARG GOPROXY

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" GOFIPS140="${GOFIPS140}" \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/gitwebhookproxy .

# distroless/static ships CA certificates and runs as uid 65532.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab

ARG VERSION=dev
ARG REVISION=unknown

LABEL org.opencontainers.image.title="gitwebhookproxy" \
      org.opencontainers.image.description="Proxy that validates Git provider webhooks and forwards them upstream" \
      org.opencontainers.image.source="https://github.com/schwichtgit/GitWebhookProxy" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.licenses="Apache-2.0"

COPY --from=builder /out/gitwebhookproxy /gitwebhookproxy

EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/gitwebhookproxy"]
