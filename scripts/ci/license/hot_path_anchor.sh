#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

fail() {
  echo "license_hot_path_anchor_gate: $*" >&2
  exit 1
}

mapfile -t hot_files < <(
  find internal/ingest internal/filter -maxdepth 1 -type f \( -name 'handler*.go' -o -name 'filter_license*.go' \) | sort
)

if [[ ${#hot_files[@]} -eq 0 ]]; then
  fail "no hot-path license files found"
fi

hot_patterns=(
  'ed25519\.Verify'
  'licensing\.VerifyJWT'
  'VerifyLicenseFile'
  '"crypto/ed25519"'
)

for file in "${hot_files[@]}"; do
  for pat in "${hot_patterns[@]}"; do
    if rg -q "$pat" "$file"; then
      fail "forbidden pattern '$pat' in $file"
    fi
  done
done

registry_file="internal/filter/registry_license.go"
if [[ ! -f "$registry_file" ]]; then
  fail "missing $registry_file"
fi

for pat in 'ed25519\.Verify' 'licensing\.VerifyJWT'; do
  if rg -q "$pat" "$registry_file"; then
    fail "forbidden pattern '$pat' in $registry_file"
  fi
done

echo "license_hot_path_anchor_gate: OK (${#hot_files[@]} hot files + registry recheck)"
