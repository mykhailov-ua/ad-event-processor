#!/usr/bin/env bash
# Role: Ensure offer version strings match deploy/vendor/offer_meta.json.
# Verify: bash scripts/ci/static/offer_version_gate.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)/scripts/lib/paths.sh"
cd "$ROOT"

META="$ROOT/deploy/vendor/offer_meta.json"
if [[ ! -f "$META" ]]; then
  printf 'offer_version_gate: missing %s\n' "$META" >&2
  exit 1
fi

VERSION="$(
  python3 - << 'PY'
import json, pathlib
print(json.loads(pathlib.Path("deploy/vendor/offer_meta.json").read_text())["version"])
PY
)"

fail=0

if ! rg -q "DefaultOfferVersion.*\"$VERSION\"" internal/trialregistry/offer_accept.go; then
  printf 'offer_version_gate: go default missing version %s in offer_accept.go\n' "$VERSION" >&2
  fail=1
fi
if ! rg -q "\"version\": \"$VERSION\"" deploy/marketing/site.config.json; then
  printf 'offer_version_gate: site.config.json missing version %s\n' "$VERSION" >&2
  fail=1
fi

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

printf 'offer_version_gate: ok version=%s\n' "$VERSION"
