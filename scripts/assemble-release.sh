#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${DIST_DIR:-$ROOT_DIR/dist}"
BUILD_DIR="${BUILD_DIR:-$ROOT_DIR/.build}"
WEB_BUILD_DIR="${WEB_BUILD_DIR:-$BUILD_DIR/web}"
RELEASES_DIR="${RELEASES_DIR:-}"
ASSET_VERSION="${ASSET_VERSION:-}"
CLIENT_VERSION="${CLIENT_VERSION:-}"
SERVER_ADDR="${SERVER_ADDR:-}"
PUBLIC_BASE_URL="${PUBLIC_BASE_URL:-}"
ASSET_BASE_URL="${ASSET_BASE_URL:-}"
RELEASE_BASE_URL="${RELEASE_BASE_URL:-}"
UPDATE_MANIFEST_URL="${UPDATE_MANIFEST_URL:-}"
UPDATE_PUBLIC_KEY="${UPDATE_PUBLIC_KEY:-}"
BUILD_COMMIT="${BUILD_COMMIT:-$(git -C "$ROOT_DIR" rev-parse --short HEAD)}"

if [[ -z "$ASSET_VERSION" ]]; then
  ASSET_VERSION="$BUILD_COMMIT"
fi
if [[ -z "$CLIENT_VERSION" ]]; then
  CLIENT_VERSION="$BUILD_COMMIT"
fi
if [[ -z "$PUBLIC_BASE_URL" ]]; then
  PUBLIC_BASE_URL="https://example.invalid"
fi
if [[ -z "$ASSET_BASE_URL" ]]; then
  ASSET_BASE_URL="$PUBLIC_BASE_URL/assets/releases/$ASSET_VERSION"
fi
if [[ -z "$RELEASE_BASE_URL" ]]; then
  RELEASE_BASE_URL="$PUBLIC_BASE_URL/releases/$CLIENT_VERSION"
fi
if [[ -z "$UPDATE_MANIFEST_URL" ]]; then
  UPDATE_MANIFEST_URL="$PUBLIC_BASE_URL/channels/stable.json"
fi

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

mkdir -p "$DIST_DIR/releases/$CLIENT_VERSION"

if [[ -n "$WEB_BUILD_DIR" && -d "$WEB_BUILD_DIR" ]]; then
  rsync -a "$WEB_BUILD_DIR/" "$DIST_DIR/"
fi
if [[ -n "$RELEASES_DIR" && -d "$RELEASES_DIR" ]]; then
  rsync -a "$RELEASES_DIR/" "$DIST_DIR/releases/$CLIENT_VERSION/"
fi

cat >"$DIST_DIR/config.js" <<EOF
window.__EO_ASSET_BASE__ = "${ASSET_BASE_URL}";
window.__EO_SERVER_ADDR__ = "${SERVER_ADDR}";
window.__EO_CLIENT_VERSION__ = "${CLIENT_VERSION}";
window.__EO_UPDATE_MANIFEST_URL__ = "${UPDATE_MANIFEST_URL}";
window.__EO_UPDATE_PUBLIC_KEY__ = "${UPDATE_PUBLIC_KEY}";
EOF

go run "$ROOT_DIR/scripts/release_metadata" \
  --asset-version "$ASSET_VERSION" \
  --asset-base-url "$ASSET_BASE_URL" \
  --build-commit "$BUILD_COMMIT" \
  --releases-dir "$DIST_DIR/releases/$CLIENT_VERSION" \
  --client-version "$CLIENT_VERSION" \
  --release-base-url "$RELEASE_BASE_URL" \
  --update-manifest-url "$UPDATE_MANIFEST_URL" \
  --release-manifest-out "$DIST_DIR/stable.json" \
  --server-addr "$SERVER_ADDR"

echo "Assembled release bundle in $DIST_DIR"
