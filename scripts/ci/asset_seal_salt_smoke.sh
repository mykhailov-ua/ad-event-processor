#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$ROOT/scripts/lib/release_garble_policy.sh"
cd "$ROOT"

if [[ -z "${ASSET_SEAL_SALT:-}" && -z "${AD_EVENT_PROCESSOR_ASSET_SEAL_SALT:-}" ]]; then
  if [[ "${RELEASE_GARBLE:-}" == "1" || "${RELEASE_ASSET_SEAL_REQUIRED:-}" == "1" ]]; then
    if [[ "${RELEASE_GARBLE_SKIP_SEED:-}" == "1" ]]; then
      echo "asset_seal_salt_smoke: skip (RELEASE_GARBLE_SKIP_SEED=1 local dev only)"
      exit 0
    fi
    echo "asset_seal_salt_smoke: ASSET_SEAL_SALT required for release builds" >&2
    exit 1
  fi
  echo "asset_seal_salt_smoke: skip (ASSET_SEAL_SALT unset)"
  exit 0
fi

export AD_EVENT_PROCESSOR_ASSET_SEAL_SALT="${AD_EVENT_PROCESSOR_ASSET_SEAL_SALT:-$ASSET_SEAL_SALT}"
go test ./internal/licensing/ -run 'DeriveReleaseAssetSealSalt|SealAsset_releaseSalt|SealAsset_wrongReleaseSalt' -count=1
echo "asset_seal_salt_smoke: OK"
