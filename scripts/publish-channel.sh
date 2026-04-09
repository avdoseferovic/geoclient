#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHANNEL_FILE="${CHANNEL_FILE:-$ROOT_DIR/dist/stable.json}"
CHANNEL_NAME="${CHANNEL_NAME:-stable.json}"
R2_REMOTE="${R2_REMOTE:-}"
R2_CHANNELS_PREFIX="${R2_CHANNELS_PREFIX:-channels}"
DRY_RUN="${DRY_RUN:-0}"

if [[ -z "$R2_REMOTE" ]]; then
  echo "R2_REMOTE is required, for example r2:geoserv-assets" >&2
  exit 1
fi
if [[ ! -f "$CHANNEL_FILE" ]]; then
  echo "missing channel file: $CHANNEL_FILE" >&2
  exit 1
fi
if ! command -v rclone >/dev/null 2>&1; then
  echo "rclone is required" >&2
  exit 1
fi

remote_base="${R2_REMOTE%/}/${R2_CHANNELS_PREFIX%/}"
remote_json="${remote_base}/${CHANNEL_NAME}"
remote_sig="${remote_json}.sig"
signature_path="${CHANNEL_FILE}.sig"

go run "$ROOT_DIR/scripts/sign_file" \
  --file "$CHANNEL_FILE" \
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

echo "uploading ${CHANNEL_NAME}"
run_rclone copyto "$CHANNEL_FILE" "$remote_json"
echo "uploading ${CHANNEL_NAME}.sig"
run_rclone copyto "$signature_path" "$remote_sig"

cat <<EOF
Published channel manifest:
  Local file: $CHANNEL_FILE
  Remote JSON: $remote_json
  Remote signature: $remote_sig
EOF
