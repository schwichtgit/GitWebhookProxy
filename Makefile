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
PACKAGES   ?= ./...

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
CHART_VALUES = $(wildcard $(CHART_DIR)/ci/*-values.yaml)
KUBE_VERSIONS ?= 1.30.0 1.33.0

# Tools come from mise.toml (run `make tools`). These two are not in the mise
# registry and are pinned here.
HELM_UNITTEST_VERSION ?= v1.2.1
# kyverno/policies, branch release-1.19
KYVERNO_POLICIES_REF  ?= 282e77f6d54e1e1a636190ff2dacc3d02c2d689c
CACHE_DIR  ?= .cache

# Trivy gates: fail on fixable HIGH/CRITICAL vulnerabilities, any secret, and
# HIGH/CRITICAL misconfigurations.
TRIVY_SEVERITY ?= HIGH,CRITICAL

comma := ,

export GOFIPS140

.DEFAULT_GOAL := help

.PHONY: help tools qa build test cover lint vuln fmt fmt-check vet tidy-check \
	docs-lint actionlint hadolint helm-lint helm-unittest kubeconform \
	kyverno-check secret-scan config-scan image docker image-scan clean
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make <target> [VAR=value]\n\nTargets:\n"} \
		/^[a-zA-Z_-]+:.*##/ {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@printf "\nVariables: VERSION=%s IMAGE=%s PLATFORMS=%s CONTAINER_CLI=%s GOFIPS140=%s\n" \
		"$(VERSION)" "$(IMAGE)" "$(PLATFORMS)" "$(CONTAINER_CLI)" "$(GOFIPS140)"

tools: ## Install the pinned toolchain from mise.toml
	mise install

qa: fmt-check vet tidy-check lint vuln docs-lint actionlint hadolint helm-lint helm-unittest kubeconform kyverno-check secret-scan config-scan ## Run all static QA gates

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

fmt-check: ## Fail if Go sources are not gofmt-formatted
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "not formatted:"; echo "$$out"; exit 1; fi

vet: ## Run go vet
	$(GO) vet ./...

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

image-scan: ## Scan the image for fixable HIGH/CRITICAL CVEs and for secrets
	trivy image --scanners vuln,secret --severity $(TRIVY_SEVERITY) --ignore-unfixed \
		--exit-code 1 $(IMAGE):$(IMAGE_TAG)

secret-scan: ## Scan the repository for committed secrets
	trivy fs --scanners secret --exit-code 1 --skip-dirs $(CACHE_DIR) --skip-dirs .claude .

config-scan: ## Scan the Dockerfile, Helm chart and workflows for misconfigurations
	trivy config --severity $(TRIVY_SEVERITY) --exit-code 1 \
		--skip-dirs $(CACHE_DIR) --skip-dirs .claude \
		--helm-set config.upstreamURL=https://example.com .

docs-lint: ## Lint and format-check markdown (markdownlint-cli2, prettier)
	markdownlint-cli2
	prettier --check '**/*.md'

actionlint: ## Lint GitHub Actions workflows (uses shellcheck)
	actionlint

hadolint: ## Lint the Dockerfile
	hadolint $(DOCKERFILE)

helm-lint: ## Lint and render the Helm chart
	helm lint --strict $(HELM_ARGS) $(CHART_DIR)
	helm template gitwebhookproxy $(CHART_DIR) $(HELM_ARGS) >/dev/null
	@for f in $(CHART_VALUES); do \
		echo "helm template -f $$f"; \
		helm template gitwebhookproxy $(CHART_DIR) -f $$f >/dev/null || exit 1; \
	done

helm-unittest: ## Run the chart unit tests (installs the helm-unittest plugin if missing)
	@helm plugin list | grep -q '^unittest' || \
		helm plugin install https://github.com/helm-unittest/helm-unittest --version $(HELM_UNITTEST_VERSION) --verify=false
	helm unittest $(CHART_DIR)

kubeconform: ## Validate rendered manifests against Kubernetes schemas
	@for k8s in $(KUBE_VERSIONS); do for f in $(CHART_VALUES); do \
		echo "Kubernetes $$k8s, values $$f"; \
		helm template gitwebhookproxy $(CHART_DIR) -f $$f --kube-version $$k8s | \
			kubeconform -strict -summary -kubernetes-version $$k8s -ignore-missing-schemas || exit 1; \
	done; done

kyverno-check: ## Check rendered manifests against Kyverno Pod Security policies
	@if [ ! -d $(CACHE_DIR)/kyverno-policies/.git ]; then \
		git clone --quiet https://github.com/kyverno/policies.git $(CACHE_DIR)/kyverno-policies; fi
	@git -C $(CACHE_DIR)/kyverno-policies fetch --quiet origin $(KYVERNO_POLICIES_REF) 2>/dev/null || true
	@git -C $(CACHE_DIR)/kyverno-policies checkout --quiet $(KYVERNO_POLICIES_REF)
	@{ kubectl kustomize $(CACHE_DIR)/kyverno-policies/pod-security/baseline; echo "---"; \
		kubectl kustomize $(CACHE_DIR)/kyverno-policies/pod-security/restricted; } > $(CACHE_DIR)/pss.yaml
	@for f in $(CHART_VALUES); do \
		echo "values $$f"; \
		helm template gitwebhookproxy $(CHART_DIR) -f $$f > $(CACHE_DIR)/rendered.yaml || exit 1; \
		kyverno apply $(CACHE_DIR)/pss.yaml --resource $(CACHE_DIR)/rendered.yaml --remove-color || exit 1; \
	done

clean: ## Remove build and test outputs
	rm -f $(BINARY) coverage.out
	rm -rf dist $(CACHE_DIR)
