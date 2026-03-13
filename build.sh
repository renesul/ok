#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

GOFLAGS="-tags stdjson"
BUILD_DIR="build"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}"
GIT_COMMIT="$(git rev-parse --short=8 HEAD 2>/dev/null || echo "dev")"
BUILD_TIME="$(date +%FT%T%z)"
GO_VERSION="$(go version | awk '{print $3}')"
APPINFO="ok/internal/appinfo"
LDFLAGS="-X ${APPINFO}.Version=${VERSION} -X ${APPINFO}.GitCommit=${GIT_COMMIT} -X ${APPINFO}.BuildTime=${BUILD_TIME} -X ${APPINFO}.GoVersion=${GO_VERSION} -s -w"

echo "=== OK Build ==="
echo "Version: ${VERSION} (${GIT_COMMIT})"
echo ""

# Kill running instances before building
if pids=$(pgrep -x "ok" 2>/dev/null); then
    echo "Stopping running ok (pid: ${pids})..."
    kill $pids 2>/dev/null || true
    sleep 0.5
fi

rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

echo "[1/2] Generating embedded files..."
CGO_ENABLED=0 go generate ./...

echo "[2/2] Building ok..."
CGO_ENABLED=0 go build ${GOFLAGS} -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/ok" .

echo ""
echo "=== Build complete ==="
ls -lh "${BUILD_DIR}"/
