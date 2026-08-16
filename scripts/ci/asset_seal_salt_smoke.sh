#!/usr/bin/env bash
# V2-D.4: verify per-release ASSET_SEAL_SALT seals and opens enterprise assets.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ -z "${ASSET_SEAL_SALT:-}" ]]; then
	echo "asset_seal_salt_smoke: skip (ASSET_SEAL_SALT unset)"
	exit 0
fi

export AD_EVENT_PROCESSOR_ASSET_SEAL_SALT="$ASSET_SEAL_SALT"
go test ./internal/licensing/ -run 'DeriveReleaseAssetSealSalt|SealAsset_releaseSalt|SealAsset_wrongReleaseSalt' -count=1
echo "asset_seal_salt_smoke: OK"
