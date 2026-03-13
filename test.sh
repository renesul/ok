#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

echo "=== OK Tests ==="
echo ""

echo "[1/3] Generating embedded files..."
CGO_ENABLED=0 go generate ./...

echo "[2/3] Running vet..."
CGO_ENABLED=0 go vet -tags stdjson ./...

echo "[3/3] Running tests..."
CGO_ENABLED=0 go test -tags stdjson -count=1 ./...

echo ""
echo "=== All tests passed ==="
