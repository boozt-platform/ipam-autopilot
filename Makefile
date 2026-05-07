# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

REPO_ROOT    := $(shell pwd)
PROVIDER_DIR := $(REPO_ROOT)/provider
LOCAL_DEV_DIR := $(REPO_ROOT)/examples/local-dev

.PHONY: help check lint lint-docker fmt test test-modules test-integration build-provider dev-setup docker-up dev-plan dev-apply dev-destroy docs docs-modules update-version

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'

## ── Pre-commit gate ─────────────────────────────────────────────────────────

check: lint fmt test build-provider docs docs-modules ## Full local gate: lint + fmt + test + docs + build (run before every commit)

## ── Code quality ────────────────────────────────────────────────────────────

lint: lint-docker ## Run all linters (Go + Dockerfile)
	cd $(REPO_ROOT)/container && golangci-lint run ./...
	cd $(REPO_ROOT)/provider  && golangci-lint run ./...

lint-docker: ## Lint Dockerfiles with hadolint
	hadolint $(REPO_ROOT)/Dockerfile
	hadolint $(REPO_ROOT)/container/Dockerfile

fmt: ## Format all Go code
	cd $(REPO_ROOT)/container && gofmt -w .
	cd $(REPO_ROOT)/provider  && gofmt -w .

## ── Tests ───────────────────────────────────────────────────────────────────

test: ## Run unit tests (container + provider + modules)
	cd $(REPO_ROOT)/container && go test ./...
	cd $(REPO_ROOT)/provider  && go test ./...
	$(MAKE) test-modules

test-modules: build-provider ## Run HCL unit tests for all modules using locally built provider
	@printf 'provider_installation {\n  dev_overrides { "boozt-platform/ipam-autopilot" = "%s" }\n  direct {}\n}\n' \
		"$(PROVIDER_DIR)" > $(REPO_ROOT)/.test.tfrc
	@for mod in $(REPO_ROOT)/modules/*/; do \
		echo "Testing $$(basename $$mod)..."; \
		cd $$mod && tofu init -upgrade -input=false -no-color > /dev/null && \
		TF_CLI_CONFIG_FILE=$(REPO_ROOT)/.test.tfrc IPAM_URL=http://localhost:8080 tofu test; \
	done
	@rm -f $(REPO_ROOT)/.test.tfrc

test-integration: ## Run integration tests (requires Docker)
	cd $(REPO_ROOT)/container && go test -tags=integration -timeout=10m ./tests/...

## ── Local development ───────────────────────────────────────────────────────

build-provider: ## Build the Terraform provider binary
	cd $(PROVIDER_DIR) && go build -o terraform-provider-ipam-autopilot .

dev-setup: build-provider docker-up ## Build provider, start docker-compose stack, and generate dev.tfrc

docker-up: ## Start docker-compose stack (rebuild if source changed)
	docker compose up --build -d


	@echo 'provider_installation {'                                           > $(LOCAL_DEV_DIR)/dev.tfrc
	@echo '  dev_overrides {'                                                >> $(LOCAL_DEV_DIR)/dev.tfrc
	@echo '    "boozt-platform/ipam-autopilot" = "$(PROVIDER_DIR)"'         >> $(LOCAL_DEV_DIR)/dev.tfrc
	@echo '  }'                                                              >> $(LOCAL_DEV_DIR)/dev.tfrc
	@echo '  direct {}'                                                      >> $(LOCAL_DEV_DIR)/dev.tfrc
	@echo '}'                                                                >> $(LOCAL_DEV_DIR)/dev.tfrc
	@echo ""
	@echo "Ready. Run:"
	@echo "  cd $(LOCAL_DEV_DIR)"
	@echo "  export TF_CLI_CONFIG_FILE=./dev.tfrc GCP_IDENTITY_TOKEN=localdev"
	@echo "  terraform plan"

dev-plan: dev-setup ## terraform plan against local docker-compose stack
	cd $(LOCAL_DEV_DIR) && TF_CLI_CONFIG_FILE=./dev.tfrc GCP_IDENTITY_TOKEN=localdev terraform plan

dev-apply: dev-setup ## terraform apply against local docker-compose stack
	cd $(LOCAL_DEV_DIR) && TF_CLI_CONFIG_FILE=./dev.tfrc GCP_IDENTITY_TOKEN=localdev terraform apply -auto-approve

dev-destroy: dev-setup ## terraform destroy local dev resources
	cd $(LOCAL_DEV_DIR) && TF_CLI_CONFIG_FILE=./dev.tfrc GCP_IDENTITY_TOKEN=localdev terraform destroy -auto-approve

## ── Documentation ───────────────────────────────────────────────────────────

docs: build-provider ## Regenerate provider docs (run from repo root)
	@SCHEMA_DIR=$$(mktemp -d) && SCHEMA=$$SCHEMA_DIR/schema.json; \
	{ \
	  echo 'provider_installation {'; \
	  echo '  dev_overrides { "boozt-platform/ipam-autopilot" = "$(PROVIDER_DIR)" }'; \
	  echo '  direct {}'; \
	  echo '}'; \
	} > $$SCHEMA_DIR/override.tfrc; \
	{ \
	  echo 'terraform {'; \
	  echo '  required_providers {'; \
	  echo '    ipam = { source = "boozt-platform/ipam-autopilot" }'; \
	  echo '  }'; \
	  echo '}'; \
	  echo 'provider "ipam" { url = "http://localhost" }'; \
	} > $$SCHEMA_DIR/main.tf; \
	TF_CLI_CONFIG_FILE=$$SCHEMA_DIR/override.tfrc $(shell which tofu 2>/dev/null || which terraform 2>/dev/null) -chdir=$$SCHEMA_DIR providers schema -json > $$SCHEMA; \
	sed -i 's|registry.opentofu.org/boozt-platform/ipam-autopilot|registry.terraform.io/hashicorp/ipam|g' $$SCHEMA; \
	cd $(PROVIDER_DIR) && go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate \
		--provider-name ipam \
		--website-source-dir templates \
		--rendered-website-dir ../docs \
		-providers-schema $$SCHEMA; \
	rm -rf $$SCHEMA_DIR

docs-modules: ## Regenerate README.md for all Terraform modules using terraform-docs
	@for mod in $(REPO_ROOT)/modules/*/; do \
		echo "Generating docs for $$(basename $$mod)..."; \
		terraform-docs --config $(REPO_ROOT)/.terraform-docs.yml $$mod; \
	done

update-version: ## Update all version references (usage: make update-version VERSION=v1.9.0)
	@test -n "$(VERSION)" || (echo "Usage: make update-version VERSION=v1.x.y" && exit 1)
	$(eval SEMVER := $(patsubst v%,%,$(VERSION)))
	$(eval MINOR  := $(word 1,$(subst ., ,$(SEMVER))).$(word 2,$(subst ., ,$(SEMVER))))
	@echo "Updating version references to $(VERSION) (constraint: ~> $(MINOR))..."
	@# Update ?ref= in .md and .tf files
	@find $(REPO_ROOT) \
		\( -name "*.md" -o -name "*.md.tmpl" -o -name "*.tf" \) \
		-not -path "*/.terraform/*" \
		-not -path "*/.git/*" \
		| xargs perl -pi -e \
		's|\?ref=v[0-9]+\.[0-9]+\.[0-9]+|?ref=$(VERSION)|g'
	@# Update provider version constraint only in files that reference boozt-platform/ipam-autopilot
	@grep -rl 'boozt-platform/ipam-autopilot' $(REPO_ROOT) \
		--include="*.tf" --include="*.md" --include="*.md.tmpl" \
		--exclude-dir=".terraform" --exclude-dir=".git" \
		| xargs perl -pi -e \
		's|version = "~> [0-9]+\.[0-9]+"|version = "~> $(MINOR)"|g'
	@echo "Done. Run 'make docs && make docs-modules' to regenerate docs."
