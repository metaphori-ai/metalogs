#!/usr/bin/env bash
set -euo pipefail

go build ./... && go vet ./...

GOBIN="${GOBIN:-$(go env GOBIN)}"
GOBIN="${GOBIN:-$(go env GOPATH)/bin}"

echo "Building and installing metalogs to ${GOBIN}..."
go install ./cmd/metalogs

echo "Done. Run: metalogs --help"
