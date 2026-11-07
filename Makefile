BINARY     ?= gitwebhookproxy
IMAGE      ?= ghcr.io/schwichtgit/gitwebhookproxy
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
# OCI tags cannot contain '+'.
IMAGE_TAG  ?= $(subst +,_,$(VERSION))

GO         ?= go
GOLANGCI_LINT ?= golangci-lint
# Build against the CMVP-validated Go Cryptographic Module (certificate #5247).
GOFIPS140  ?= v1.0.0
GOPROXY    ?=
LDFLAGS    ?= -s -w -X main.version=$(VERSION)

COVER_MIN  ?= 85
PACKAGES   ?= $(shell $(GO) list ./... | grep -v /examples)

DOCKERFILE ?= build/package/Dockerfile
# Single platform by default so the image can be loaded locally.
# Set PLATFORMS=linux/amd64,linux/arm64 for a multi-arch build (requires PUSH=1 with docker).
PLATFORMS  ?= linux/$(shell $(GO) env GOARCH)
PUSH       ?= 0

# Prefer docker buildx, fall back to podman.
CONTAINER_CLI ?= $(shell if docker buildx version >/dev/null 2>&1; then echo docker; elif command -v podman >/dev/null 2>&1; then echo podman; else echo docker; fi)

CHART_DIR  ?= deployments/kubernetes/chart/gitwebhookproxy
# Required chart values for lint and template.
HELM_ARGS  ?= --set config.upstreamURL=https://example.com

comma := ,

export GOFIPS140

.DEFAULT_GOAL := help

.PHONY: help build test cover lint vuln fmt tidy-check image docker image-scan helm-lint clean

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make <target> [VAR=value]\n\nTargets:\n"} \
		/^[a-zA-Z_-]+:.*##/ {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@printf "\nVariables: VERSION=%s IMAGE=%s PLATFORMS=%s CONTAINER_CLI=%s GOFIPS140=%s\n" \
		"$(VERSION)" "$(IMAGE)" "$(PLATFORMS)" "$(CONTAINER_CLI)" "$(GOFIPS140)"

build: ## Build the binary
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

test: ## Run tests with the race detector
	$(GO) test -race $(PACKAGES)

cover: ## Run tests with coverage; fail below COVER_MIN percent
	$(GO) test -race -covermode=atomic -coverprofile=coverage.out $(PACKAGES)
	@total=$$($(GO) tool cover -func=coverage.out | awk '/^total:/ {sub("%", "", $$3); print $$3}'); \
	echo "total coverage: $${total}% (minimum $(COVER_MIN)%)"; \
	awk -v t="$${total}" -v m="$(COVER_MIN)" 'BEGIN {exit !(t >= m)}' || \
		{ echo "coverage below $(COVER_MIN)%"; exit 1; }

lint: ## Run golangci-lint
	$(GOLANGCI_LINT) run

vuln: ## Check dependencies for known vulnerabilities
	govulncheck ./...

fmt: ## Format Go sources
	$(GO) fmt ./...

tidy-check: ## Fail if go.mod or go.sum are not tidy
	$(GO) mod tidy -diff

image: ## Build the container image (PLATFORMS, PUSH=1 to push)
ifeq ($(CONTAINER_CLI),podman)
	podman build \
		--platform $(PLATFORMS) \
		--manifest $(IMAGE):$(IMAGE_TAG) \
		--build-arg VERSION=$(VERSION) \
		--build-arg REVISION=$(COMMIT) \
		--build-arg GOFIPS140=$(GOFIPS140) \
		--build-arg GOPROXY=$(GOPROXY) \
		-f $(DOCKERFILE) .
	@if [ "$(PUSH)" = "1" ]; then podman manifest push --all $(IMAGE):$(IMAGE_TAG) docker://$(IMAGE):$(IMAGE_TAG); fi
else
	docker buildx build \
		--platform $(PLATFORMS) \
		--tag $(IMAGE):$(IMAGE_TAG) \
		--build-arg VERSION=$(VERSION) \
		--build-arg REVISION=$(COMMIT) \
		--build-arg GOFIPS140=$(GOFIPS140) \
		--build-arg GOPROXY=$(GOPROXY) \
		$(if $(filter 1,$(PUSH)),--push,$(if $(findstring $(comma),$(PLATFORMS)),,--load)) \
		-f $(DOCKERFILE) .
endif

docker: image ## Alias for image

image-scan: ## Scan the image for fixable HIGH/CRITICAL vulnerabilities
	trivy image --severity HIGH,CRITICAL --ignore-unfixed --exit-code 1 $(IMAGE):$(IMAGE_TAG)

helm-lint: ## Lint and render the Helm chart
	helm lint --strict $(HELM_ARGS) $(CHART_DIR)
	helm template gitwebhookproxy $(CHART_DIR) $(HELM_ARGS) >/dev/null
	@for f in $(wildcard $(CHART_DIR)/ci/*-values.yaml); do \
		echo "helm template -f $$f"; \
		helm template gitwebhookproxy $(CHART_DIR) -f $$f >/dev/null || exit 1; \
	done

clean: ## Remove build and test outputs
	rm -f $(BINARY) coverage.out
	rm -rf dist
