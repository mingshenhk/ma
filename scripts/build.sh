#!/bin/sh
set -eu
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT/frontend"
npm ci
npm run build
cd "$ROOT/backend"
go test ./...
go vet ./...
mkdir -p "$ROOT/bin"
go build -trimpath -o "$ROOT/bin/haojiayuan" ./cmd/haojiayuan
