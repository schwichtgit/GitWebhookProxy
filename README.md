# Git Webhook Proxy

A proxy service for Git webhooks that validates and forwards webhook events to an upstream URL.

**Go Version:** 1.27.3
**FIPS Support:** Yes (when built with `GOFIPS140='v1.0.0'`)

## Overview

Git Webhook Proxy is a lightweight service that sits between Git providers (GitHub, GitLab) and your internal services. It:

- Validates webhook signatures/tokens
- Filters requests based on allowed paths
- Filters requests based on users (ignored or allowed)
- Forwards valid requests to an upstream URL

## Features

- Support for GitHub and GitLab webhooks
- Webhook signature/token validation
- Path-based filtering
- User-based filtering
- Health check endpoint

## Security Improvements

This version has been updated to use modern, well-maintained Go modules and includes FIPS 140-3 compliance:

- Replaced `github.com/julienschmidt/httprouter` with `github.com/go-chi/chi/v5`
- Replaced `github.com/namsral/flag` with `github.com/spf13/pflag`
- Added FIPS 140-3 compliant mode (when built with `GOFIPS140='v1.0.0'`)
- Updated to Go 1.27.3

These changes address security concerns related to using outdated or abandoned modules that might have unpatched CVEs and provide FIPS 140-3 compliance for environments that require it.

## Usage

### Command-line Arguments

| Flag | Description | Default |
|------|-------------|---------|
| `--listen` | Address on which the proxy listens | `:8080` |
| `--upstreamURL` | URL to which the proxy requests will be forwarded | (required) |
| `--provider` | Git provider which generates the webhook (github or gitlab) | `github` |
| `--secret` | Secret of the webhook API. If not set, validation is not performed | `""` |
| `--allowedPaths` | Comma-separated list of allowed paths | `""` (all paths allowed) |
| `--ignoredUsers` | Comma-separated list of users to ignore | `""` |
| `--allowedUsers` | Comma-separated list of users to allow | `""` (all users allowed) |

### Environment Variables

All command-line arguments can also be specified as environment variables with the `GWP_` prefix. For example:

```
GWP_LISTEN=:8080
GWP_UPSTREAMURL=http://jenkins.example.com
GWP_PROVIDER=github
GWP_SECRET=your-webhook-secret
```

### Example

```bash
# Run with command-line arguments
./gitwebhookproxy --listen=:8080 --upstreamURL=http://jenkins.example.com --provider=github --secret=your-webhook-secret

# Run with environment variables
export GWP_UPSTREAMURL=http://jenkins.example.com
export GWP_SECRET=your-webhook-secret
./gitwebhookproxy
```

### Docker

```bash
# Standard run
docker run -p 8080:8080 \
  -e GWP_UPSTREAMURL=http://jenkins.example.com \
  -e GWP_PROVIDER=github \
  -e GWP_SECRET=your-webhook-secret \
  stakater/gitwebhookproxy:latest


## Building from Source

### Standard Build

```bash
# Clone the repository
git clone https://github.com/stakater/GitWebhookProxy.git
cd GitWebhookProxy

# Build the binary
go build -o gitwebhookproxy

# Run the binary
./gitwebhookproxy --upstreamURL=http://jenkins.example.com
```

### FIPS-Compliant Build

FIPS 140-3 mode is on by default in the container image and in `make build`. Both set
`GOFIPS140=v1.0.0`, which builds against the CMVP-validated Go Cryptographic Module
(certificate #5247). See https://go.dev/doc/security/fips140.

```bash
# Local build with FIPS 140-3 mode
GOFIPS140=v1.0.0 go build -o gitwebhookproxy-fips

# Container build (override with --build-arg GOFIPS140=off to disable)
docker build -f build/package/Dockerfile.multi -t gitwebhookproxy .
```

A binary built with `GOFIPS140` set:
- Uses the selected Go Cryptographic Module version
- Runs FIPS 140-3 mode by default (no runtime setting needed)
- Restricts `crypto/tls` to approved protocol versions, cipher suites, signature algorithms and key exchanges
- Logs `FIPS 140-3 mode enabled (Go Cryptographic Module v1.0.0...)` at startup

A binary built without `GOFIPS140` can still enable the mode at runtime with
`GODEBUG=fips140=on`.

## Health Check

The proxy provides a health check endpoint at `/health` that can be used to monitor the service.

```bash
curl http://localhost:8080/health
```

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.
