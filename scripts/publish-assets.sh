#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ASSET_VERSION="${ASSET_VERSION:-}"
ASSET_PUBLIC_BASE="${ASSET_PUBLIC_BASE:-}"
R2_REMOTE="${R2_REMOTE:-}"
R2_RELEASES_PREFIX="${R2_RELEASES_PREFIX:-releases}"
ASSET_SOURCE_DIR="${ASSET_SOURCE_DIR:-$ROOT_DIR}"
ASSET_DIRS="${ASSET_DIRS:-gfx maps pub mfx sfx}"
BUILD_COMMIT="${BUILD_COMMIT:-$(git -C "$ROOT_DIR" rev-parse --short HEAD)}"
DRY_RUN="${DRY_RUN:-0}"

if [[ -z "$ASSET_VERSION" ]]; then
  echo "ASSET_VERSION is required" >&2
  exit 1
fi
if [[ -z "$ASSET_PUBLIC_BASE" ]]; then
  echo "ASSET_PUBLIC_BASE is required, for example https://assets.geoserv.app/releases" >&2
  exit 1
fi
if [[ -z "$R2_REMOTE" ]]; then
  echo "R2_REMOTE is required, for example r2:geoserv-assets" >&2
  exit 1
fi
if ! command -v rclone >/dev/null 2>&1; then
  echo "rclone is required" >&2
  exit 1
fi

release_base_url="${ASSET_PUBLIC_BASE%/}/$ASSET_VERSION"
release_remote="${R2_REMOTE%/}/${R2_RELEASES_PREFIX%/}/$ASSET_VERSION"
work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT

manifest_path="$work_dir/manifest.json"
signature_path="$manifest_path.sig"

go run "$ROOT_DIR/scripts/release_metadata" \
  --assets-dir "$ASSET_SOURCE_DIR" \
  --asset-version "$ASSET_VERSION" \
  --asset-base-url "$release_base_url" \
  --build-commit "$BUILD_COMMIT" \
  --asset-manifest-out "$manifest_path"

go run "$ROOT_DIR/scripts/sign_file" \
  --file "$manifest_path" \
  --out "$signature_path"

rclone_flags=()
if [[ "$DRY_RUN" == "1" ]]; then
  rclone_flags+=(--dry-run)
fi

run_rclone() {
  if [[ ${#rclone_flags[@]} -gt 0 ]]; then
    rclone "$@" "${rclone_flags[@]}"
    return
  fi
  rclone "$@"
}

for dir in $ASSET_DIRS; do
  source_path="$ASSET_SOURCE_DIR/$dir"
  if [[ ! -d "$source_path" ]]; then
    echo "missing asset directory: $source_path" >&2
    exit 1
  fi
  echo "syncing $dir -> $release_remote/$dir"
  run_rclone sync "$source_path" "$release_remote/$dir"
done

echo "uploading manifest.json"
run_rclone copyto "$manifest_path" "$release_remote/manifest.json"
echo "uploading manifest.json.sig"
run_rclone copyto "$signature_path" "$release_remote/manifest.json.sig"

cat <<EOF
Published asset release:
  Version: $ASSET_VERSION
  Public base: $release_base_url
  Remote path: $release_remote

Files uploaded:
  $(printf '%s\n  ' $ASSET_DIRS)manifest.json
  manifest.json.sig
EOF
