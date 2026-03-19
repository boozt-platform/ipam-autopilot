# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

REPO_ROOT := $(shell pwd)
PROVIDER_DIR := $(REPO_ROOT)/provider
PROVIDER_BIN := $(PROVIDER_DIR)/terraform-provider-ipam-autopilot
LOCAL_DEV_DIR := $(REPO_ROOT)/examples/local-dev

.PHONY: build-provider dev-setup dev-plan dev-apply dev-destroy

## Build the Terraform provider binary
build-provider:
	cd $(PROVIDER_DIR) && go build -o terraform-provider-ipam-autopilot .

## Generate dev.tfrc and prepare local dev environment
dev-setup: build-provider
	@echo 'provider_installation {' > $(LOCAL_DEV_DIR)/dev.tfrc
	@echo '  dev_overrides {' >> $(LOCAL_DEV_DIR)/dev.tfrc
	@echo '    "boozt-platform/ipam-autopilot" = "$(PROVIDER_DIR)"' >> $(LOCAL_DEV_DIR)/dev.tfrc
	@echo '  }' >> $(LOCAL_DEV_DIR)/dev.tfrc
	@echo '  direct {}' >> $(LOCAL_DEV_DIR)/dev.tfrc
	@echo '}' >> $(LOCAL_DEV_DIR)/dev.tfrc
	@echo ""
	@echo "Ready. Run:"
	@echo "  cd $(LOCAL_DEV_DIR)"
	@echo "  export TF_CLI_CONFIG_FILE=./dev.tfrc GCP_IDENTITY_TOKEN=localdev"
	@echo "  terraform plan"

## Run terraform plan against local docker-compose stack
dev-plan: dev-setup
	cd $(LOCAL_DEV_DIR) && TF_CLI_CONFIG_FILE=./dev.tfrc GCP_IDENTITY_TOKEN=localdev terraform plan

## Run terraform apply against local docker-compose stack
dev-apply: dev-setup
	cd $(LOCAL_DEV_DIR) && TF_CLI_CONFIG_FILE=./dev.tfrc GCP_IDENTITY_TOKEN=localdev terraform apply -auto-approve

## Destroy all local dev resources
dev-destroy: dev-setup
	cd $(LOCAL_DEV_DIR) && TF_CLI_CONFIG_FILE=./dev.tfrc GCP_IDENTITY_TOKEN=localdev terraform destroy -auto-approve
