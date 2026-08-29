#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

PATTERN='(\bespx\b|\bESPX_[A-Z0-9_]*\b|BidShard|\bbidshard\b|X-BidShard)'

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
    echo "check_no_legacy_naming: forbidden in ${label}:"
    printf '  %s\n' "${hits[@]}"
    exit 1
  fi
}

bureaucratic_check() {
  local label="$1"
  shift
  local hits=()
  while IFS= read -r line; do
    [[ -n "$line" ]] && hits+=("$line")
  done < <(rg -n '\bGAP-[A-Z][A-Z0-9]*-[0-9]+\b' "$@" 2> /dev/null || true)

  if ((${#hits[@]} > 0)); then
    echo "check_no_legacy_naming: bureaucratic ticket ID in ${label} (use semantic slug; naming.mdc):"
    printf '  %s\n' "${hits[@]}"
    exit 1
  fi
}

milestone_check() {
  local label="$1"
  shift
  local pattern='(\bCPA-M[0-9]+\b|\bGM-M[0-9]+\b|\bRP-M[0-9]+\b|\bGMM4\b|\bPS-[GH][0-9]{2}\b|\bL1CIDRBlockEnabled\b|\bL15ProxyVPNBlockEnabled\b|\bIPv6RotationL1Enabled\b|\bIPv4RotationL1Enabled\b|\bTLSFingerprintL1Enabled\b|\bwriteGnetSafeViewL15\b|\bl15HookHandler\b|\bl1HookHandler\b)'
  local hits=()
  while IFS= read -r line; do
    [[ -n "$line" ]] && hits+=("$line")
  done < <(rg -n "$pattern" "$@" 2> /dev/null || true)

  if ((${#hits[@]} > 0)); then
    echo "check_no_legacy_naming: milestone or gap token in ${label} (use semantic slug; naming.mdc):"
    printf '  %s\n' "${hits[@]}"
    exit 1
  fi
}

strict_check "docs/ARCHITECTURE.md" "$ROOT/docs/ARCHITECTURE.md"
strict_check "docs/DEVELOPMENT.md" "$ROOT/docs/DEVELOPMENT.md"
strict_check ".cursor/rules/LICENSING.mdc" "$ROOT/.cursor/rules/LICENSING.mdc"
if [[ -d "$ROOT/web/src" ]]; then
  strict_check "web/src" "$ROOT/web/src"
fi
if [[ -d "$ROOT/web/e2e" ]]; then
  strict_check "web/e2e" "$ROOT/web/e2e" --glob '!**/node_modules/**'
fi

bureaucratic_check "README.md" "$ROOT/README.md"
bureaucratic_check "docs/" "$ROOT/docs"
bureaucratic_check "deploy/vendor/" "$ROOT/deploy/vendor"
bureaucratic_check ".cursor/rules/" "$ROOT/.cursor/rules" --glob '*.mdc'

milestone_check "internal/" "$ROOT/internal"
if [[ -d "$ROOT/web/src" ]]; then
  milestone_check "web/src" "$ROOT/web/src"
fi
if [[ -d "$ROOT/web/e2e" ]]; then
  milestone_check "web/e2e" "$ROOT/web/e2e" --glob '!**/node_modules/**'
fi
milestone_check ".env.example" "$ROOT/.env.example"

is_allowlisted() {
  local rel="$1"
  local prefix
  for prefix in "${ALLOWED_PREFIXES[@]}"; do
    if [[ "$rel" == "$prefix" || "$rel" == "$prefix"* ]]; then
      return 0
    fi
  done

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
  echo "check_no_legacy_naming: forbidden outside allowlist (${#violations[@]} hit(s)):"
  printf '  %s\n' "${violations[@]}"
  exit 1
fi

for rule in "$ROOT/.cursor/rules"/*.mdc; do
  base="$(basename "$rule" .mdc)"
  upper="$(printf '%s' "$base" | tr '[:lower:]' '[:upper:]')"
  if [[ "$base" != "$upper" ]]; then
    echo "check_no_legacy_naming: .cursor/rules basename must be UPPERCASE: $rule" >&2
    exit 1
  fi
done

receiver_check() {
  local hits=()
  while IFS= read -r line; do
    [[ -n "$line" ]] && hits+=("$line")
  done < <(rg -n '^func \([a-z]{3,} ' \
    internal/ pkg/ cmd/ \
    --glob '*.go' \
    --glob '!*_test.go' \
    2> /dev/null || true)

  if ((${#hits[@]} > 0)); then
    echo "check_no_legacy_naming: method receiver must be 1-2 letters (quality.mdc):"
    printf '  %s\n' "${hits[@]}"
    exit 1
  fi
}

receiver_check

echo "check_no_legacy_naming: ok"
