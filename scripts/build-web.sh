#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WEB_DIR="${WEB_DIR:-$ROOT_DIR/web}"
WASM_EXEC_SRC="$(go env GOROOT)/lib/wasm/wasm_exec.js"

if [[ ! -f "$WASM_EXEC_SRC" ]]; then
  WASM_EXEC_SRC="$(go env GOROOT)/misc/wasm/wasm_exec.js"
fi

cp "$WASM_EXEC_SRC" "$WEB_DIR/wasm_exec.js"
GOOS=js GOARCH=wasm go build -o "$WEB_DIR/eoclient.wasm" ./cmd/eoclient

echo "Built $WEB_DIR/eoclient.wasm"
