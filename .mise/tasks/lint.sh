#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
#MISE description="Run linters (go fmt check + go vet)"
set -euo pipefail

echo "Running go fmt..."
UNFORMATTED=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./third_party/*" -exec gofmt -l {} +)
if [ -n "$UNFORMATTED" ]; then
  echo "Unformatted files:"
  echo "$UNFORMATTED"
  exit 1
fi

echo "Running go vet..."
go vet -mod=mod ./...
echo "✓ Linting passed"
