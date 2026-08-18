#!/usr/bin/env bash
# Fail on legacy espx naming outside the migration allowlist.
# Strict (no allowlist): public docs + admin web sources.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

PATTERN='(\bespx\b|ESPX_|/run/espx|/etc/espx|eSPX)'

# Prefix allowlist: paths still migrating (see docs/NAMING.md §7).
ALLOWED_PREFIXES=(
  .internal-migration-stub
  internal/
  pkg/
  cmd/
  deploy/
  scripts/
  api/
  .github/
  docs/
  tests/
  web/e2e/node_modules/
)

# Root files still migrating (shrink as renames land).
ALLOWED_ROOT_FILES=(
  go.mod
  go.sum
  Makefile
  Taskfile.yaml
  buf.yaml
  .env.example
)

strict_check() {
  local label="$1"
  shift
  local hits=()
  while IFS= read -r line; do
    [[ -n "$line" ]] && hits+=("$line")
  done < <(rg -n "$PATTERN" "$@" 2> /dev/null || true)
  if ((${#hits[@]} > 0)); then
    echo "check_no_espx: forbidden in ${label}:"
    printf '  %s\n' "${hits[@]}"
    exit 1
  fi
}

strict_check "README.md" "$ROOT/README.md"
strict_check "docs/QUICKSTART.md" "$ROOT/docs/QUICKSTART.md"
strict_check "docs/LICENSE.md" "$ROOT/docs/LICENSE.md"
strict_check "web/src" "$ROOT/web/src"
strict_check "web/e2e" "$ROOT/web/e2e" --glob '!**/node_modules/**'

is_allowlisted() {
  local rel="$1"
  local prefix
  for prefix in "${ALLOWED_PREFIXES[@]}"; do
    if [[ "$rel" == "$prefix" || "$rel" == "$prefix"* ]]; then
      return 0
    fi
  done
  local base="${rel##*/}"
  local allowed
  for allowed in "${ALLOWED_ROOT_FILES[@]}"; do
    if [[ "$rel" == "$allowed" ]]; then
      return 0
    fi
  done
  return 1
}

violations=()
while IFS= read -r hit; do
  [[ -z "$hit" ]] && continue
  file="${hit%%:*}"
  rel="${file#"$ROOT"/}"
  if is_allowlisted "$rel"; then
    continue
  fi
  violations+=("$hit")
done < <(rg -n "$PATTERN" "$ROOT" \
  --glob '!.git/**' \
  --glob '!**/node_modules/**' \
  --glob '!web/dist/**' \
  --glob '!bin/**' \
  --glob '!*.cover' \
  --glob '!*.out' \
  2> /dev/null || true)

if ((${#violations[@]} > 0)); then
  echo "check_no_espx: forbidden outside allowlist (${#violations[@]} hit(s)):"
  printf '  %s\n' "${violations[@]}"
  exit 1
fi

echo "check_no_espx: ok"
