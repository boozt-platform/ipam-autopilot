#!/usr/bin/env bash
set -euo pipefail

echo "Installing development tools..."

# golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
  | sh -s -- -b "$(go env GOPATH)/bin"

# go tools
go install golang.org/x/tools/cmd/goimports@latest

echo "All tools installed."
echo ""
echo "Available make targets:"
echo "  make dev-apply    — build provider + terraform apply against docker-compose"
echo "  make dev-destroy  — terraform destroy"
echo "  make build-provider — build provider binary"
