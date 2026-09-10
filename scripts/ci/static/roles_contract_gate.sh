#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

perm_file="deploy/operator/permissions.yaml"
catalog_file="internal/access/catalog.go"

if [[ ! -f "$perm_file" ]]; then
  echo "roles_contract_gate: missing $perm_file"
  exit 1
fi

if [[ ! -f "$catalog_file" ]]; then
  echo "roles_contract_gate: missing $catalog_file"
  exit 1
fi

missing=0
while IFS= read -r line; do
  id="${line#  - id: }"
  [[ "$line" == *"id:"* ]] || continue
  if ! grep -Fq "$id" "$catalog_file"; then
    echo "roles_contract_gate: permission $id missing from access catalog"
    missing=1
  fi
done < "$perm_file"

if [[ "$missing" -ne 0 ]]; then
  exit 1
fi

echo "roles_contract_gate: ok"
