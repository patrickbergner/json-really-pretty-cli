#!/usr/bin/env bash
# Cross compiles jrp for every supported platform into dist/.
# Usage: ./build.sh [optional version override]
set -euo pipefail

cd "$(dirname "$0")"

VERSION="${1:-$(cat VERSION)}"
OUT_DIR="dist"

echo "formatting"
go fmt ./...

echo "vetting"
go vet ./...

echo "testing"
go test ./...

TARGETS=(
  "windows/amd64"
  "windows/arm64"
  "linux/amd64"
  "linux/arm64"
  "darwin/arm64"
)

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

for target in "${TARGETS[@]}"; do
  goos="${target%/*}"
  goarch="${target#*/}"

  platform="$goos"
  if [[ "$goos" == "darwin" ]]; then
    platform="macos"
  fi

  out="$OUT_DIR/jrp_${platform}_${goarch}"
  if [[ "$goos" == "windows" ]]; then
    out="${out}.exe"
  fi

  echo "building $out"
  # CGO is off so every target is a plain cross compile with no toolchain of its own; -trimpath keeps local paths out of the binary.
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o "$out" .
done

echo "built version ${VERSION}, done"
