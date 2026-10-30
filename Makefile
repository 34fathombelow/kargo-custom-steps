REGISTRY ?= ghcr.io/34fathombelow
TOOLS    := $(notdir $(wildcard docker/*))
STEPS    := $(sort $(basename $(notdir $(wildcard kargo/custom-steps/*/*/*.yaml))))
RUNSTEP  := go -C test run ./cmd/runstep

.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-14s %s\n", $$1, $$2}'

##@ Images

.PHONY: images
images: ## Build step images locally (all, or TOOL=<tool>)
	@for t in $(or $(TOOL),$(TOOLS)); do \
		tag=$$(scripts/image-tag.sh $$t) || exit 1; \
		echo "==> $(REGISTRY)/$$t:$$tag"; \
		docker build -q -t $(REGISTRY)/$$t:$$tag docker/$$t || exit 1; \
	done

.PHONY: push
push: ## Build multi-arch images and push them to $(REGISTRY) (all, or TOOL=<tool>; needs docker login)
	@for t in $(or $(TOOL),$(TOOLS)); do \
		tag=$$(scripts/image-tag.sh $$t) || exit 1; \
		echo "==> $(REGISTRY)/$$t:$$tag (linux/amd64, linux/arm64)"; \
		docker buildx build --platform linux/amd64,linux/arm64 --push -t $(REGISTRY)/$$t:$$tag docker/$$t || exit 1; \
	done

##@ Fork

.PHONY: fork
fork: ## Retarget repo URLs, image registry, shard and Argo CD cluster at your fork (see scripts/fork.sh --help)
	scripts/fork.sh $(ARGS)

##@ Test

.PHONY: test
test: ## Run step fixtures with the local harness (all, or STEP=<step>)
	@fail=0; for s in $(or $(STEP),$(STEPS)); do \
		step=$$(ls kargo/custom-steps/*/$$s/$$s.yaml); \
		for f in test/fixtures/$$s/*.yaml; do \
			[ -e "$$f" ] || { echo "no fixtures for $$s"; fail=1; continue; }; \
			$(RUNSTEP) -repo $(CURDIR) -step $(CURDIR)/$$step -fixture $(CURDIR)/$$f $(if $(V),-v) || fail=1; \
		done; \
	done; exit $$fail

.PHONY: test-services
test-services: ## Start the services some fixtures call (flyway-migrate's Postgres, cluster-version's fake Argo CD metrics, sentry-release's fake Sentry)
	test/fixtures/flyway-migrate/test-db.sh
	test/fixtures/cluster-version/fake-api.sh
	test/fixtures/sentry-release/fake-api.sh

.PHONY: test-services-down
test-services-down: ## Remove the fixture services
	test/fixtures/flyway-migrate/test-db.sh down
	test/fixtures/cluster-version/fake-api.sh down
	test/fixtures/sentry-release/fake-api.sh down

.PHONY: step-images
step-images: ## Refresh the "Image" section of every step README from its Dockerfile
	scripts/step-images.sh

.PHONY: lint
lint: ## Check kargo/ and the step conventions
	go -C test run ./cmd/lint -repo $(CURDIR)

.PHONY: validate
validate: ## Validate kargo/ against the Kargo CRD schemas (offline; KARGO_VERSION=v1.11.8)
	scripts/validate-schemas.sh

.PHONY: check
check: lint validate test ## lint + validate + test

##@ Release

.PHONY: release-assets
release-assets: ## Build the release assets into dist/, as release.yaml does (VERSION=<tag> for its links)
	scripts/release-assets.sh $(VERSION)

##@ Fork settings

.PHONY: set-shard
set-shard: ## Point every demo Stage/Warehouse at another shard: SHARD=<name>
	@test -n "$(SHARD)" || { echo "usage: make set-shard SHARD=<name>"; exit 1; }
	scripts/fork.sh --shard $(SHARD)
