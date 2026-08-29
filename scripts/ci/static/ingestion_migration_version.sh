#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"

dir="$ROOT/internal/ingest/migrations"
fail=0

while IFS= read -r version; do
  [[ -z "$version" ]] && continue
  mapfile -t files < <(compgen -G "$dir/${version}_*.sql" || true)
  if ((${#files[@]} > 1)); then
    echo "ingestion_migration_version_gate: duplicate version $version:" >&2
    printf '  %s\n' "${files[@]#$dir/}" >&2
    fail=1
  fi
done < <(find "$dir" -maxdepth 1 -name '*.sql' -printf '%f\n' | sed 's/^\([0-9]\{5\}\)_.*/\1/' | sort | uniq -d)

if ((fail != 0)); then
  exit 1
fi

echo "ingestion_migration_version_gate: ok"
