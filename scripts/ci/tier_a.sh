#!/usr/bin/env bash
set -euo pipefail

# Role: Tier-A repo hygiene: docs layout, version tags, HTML ban in controlplane, brand boundary.
# Execution context: CI merge-pr-fast (via pr_fast) and local; scans tree plus optional diff vs BASE.
# Invariants/contracts enforced: Fail-closed on forbidden docs paths, M-number tags, HTMX/HTML in cold handlers.
# Verify: bash scripts/ci/tier_a.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

fail=0

echo "tier_a: check sources-only runtime artifacts..."
if [[ -e license.jwt ]]; then
  echo "sources-only: repo-root license.jwt must not exist (use var/license.jwt for local dev)" >&2
  fail=1
fi

echo "tier_a: check docs layout..."
for forbidden in docs/COMPLIANCE_MATRIX.md docs/MILESTONE.md docs/SELF_HOSTED.md docs/PROTECTION.md; do
  if [[ -f "$forbidden" ]]; then
    echo "docs layout: $forbidden must not exist" >&2
    fail=1
  fi
done

for required in docs/ARCHITECTURE.md docs/DEVELOPMENT.md; do
  if [[ ! -f "$required" ]]; then
    echo "docs layout: missing $required" >&2
    fail=1
  fi
done

allowed_docs=(
  ARCHITECTURE.md
  DEVELOPMENT.md
  INTEGRATIONS.md
)

for path in docs/*.md; do
  [[ -e "$path" ]] || continue
  base="$(basename "$path")"
  ok=0
  for name in "${allowed_docs[@]}"; do
    if [[ "$base" == "$name" ]]; then
      ok=1
      break
    fi
  done
  if [[ "$ok" -eq 0 ]]; then
    echo "docs layout: unexpected docs/$base" >&2
    fail=1
  fi
done

if [[ -d docs/runbooks ]]; then
  echo "docs layout: docs/runbooks/ must not exist (content in .cursor/rules/*.mdc)" >&2
  fail=1
fi

if [[ -d fraudtrain ]]; then
  echo "docs layout: fraudtrain/ at repo root must not exist" >&2
  fail=1
fi

if [[ -d examples ]] || [[ -d examples/fraudtrain ]]; then
  echo "docs layout: examples/ must not exist (use model/)" >&2
  fail=1
fi

if [[ -d testdata ]]; then
  echo "docs layout: testdata/ at repo root must not exist (generate via make fraudtrain-check)" >&2
  fail=1
fi

echo "tier_a: check no version-tag references..."
pattern_version_tag='\bM[0-9]+'
scan_version_tag() {
  local path="$1"
  if rg -n "$pattern_version_tag" "$path" > /dev/null 2>&1; then
    echo "check_no_version_tag_refs: version tag in $path"
    rg -n "$pattern_version_tag" "$path" || true
    fail=1
  fi
}

while IFS= read -r -d '' file; do
  case "$file" in
    */pb/* | */sqlc/* | *_test.go | */testdata/*) continue ;;
  esac
  case "$(basename "$file")" in
    *.pb.go | *_grpc.pb.go | *_vtproto.pb.go | *_bpfel.go | *_bpfeb.go) continue ;;
  esac
  scan_version_tag "$file"
done < <(find internal cmd pkg tests -name '*.go' -print0 2> /dev/null || true)

while IFS= read -r -d '' file; do
  scan_version_tag "$file"
done < <(find scripts -name '*.sh' -print0 2> /dev/null || true)

BASE="${1:-origin/main}"
if git rev-parse --verify "$BASE" > /dev/null 2>&1; then
  mapfile -t diff_hits < <(
    git diff "$BASE"...HEAD -- '*.go' '*.sh' \
      | rg "$pattern_version_tag" \
      | rg -v 'versionTag|versionWord|check_no_version_tag_refs' || true
  )
  if ((${#diff_hits[@]} > 0)); then
    echo "check_no_version_tag_refs: forbidden version tag in diff:"
    printf '  %s\n' "${diff_hits[@]}"
    fail=1
  fi
fi

echo "tier_a: check no HTML/HTMX in controlplane/payment..."
check_html_dir() {
  local dir="$1"
  [[ -d "$dir" ]] || return 0
  while IFS= read -r -d '' file; do
    case "$file" in
      *_test.go | */testdata/*) continue ;;
      */admin_ui_static.go) continue ;;
    esac
    if rg -n 'Set\("Content-Type",\s*"text/html' "$file" > /dev/null 2>&1; then
      echo "check_no_html_success: text/html success Content-Type in $file"
      rg -n 'Set\("Content-Type",\s*"text/html' "$file" || true
      fail=1
    fi
    if rg -n 'HX-Request|hx-post|hx-get|WriteHTMX|HTMXError' "$file" > /dev/null 2>&1; then
      echo "check_no_html_success: HTMX reference in $file"
      rg -n 'HX-Request|hx-post|hx-get|WriteHTMX|HTMXError' "$file" || true
      fail=1
    fi
  done < <(find "$dir" -name '*.go' -print0)
}

check_html_dir internal/controlplane
check_html_dir internal/payment

if rg -n 'text/template' internal/controlplane internal/payment --glob '*.go' --glob '!*_test.go' > /dev/null 2>&1; then
  echo "check_no_html_success: text/template import in management/payment"
  rg -n 'text/template' internal/controlplane internal/payment --glob '*.go' --glob '!*_test.go' || true
  fail=1
fi

echo "tier_a: check no slog in service files..."
mapfile -t slog_hits < <(
  rg -n 'slog\.' internal/controlplane/service_*.go \
    --glob '!internal/controlplane/service.go' 2> /dev/null || true
)

if ((${#slog_hits[@]} > 0)); then
  echo "check_no_service_slog: slog forbidden in service_*.go (except service.go lifecycle):"
  printf '  %s\n' "${slog_hits[@]}"
  fail=1
fi

echo "tier_a: check brand boundary..."
pattern_brand='ad-event-processor\.com'
scan_brand() {
  local path="$1"
  case "$path" in
    */pb/* | */sqlc/* | *_test.go | */testdata/*) return 0 ;;
  esac
  case "$(basename "$path")" in
    *.pb.go | *_grpc.pb.go | *_vtproto.pb.go | *_bpfel.go | *_bpfeb.go) return 0 ;;
  esac
  local hits
  hits="$(rg -n "$pattern_brand" "$path" 2> /dev/null | rg -v 'ad-event-processor' || true)"
  if [[ -n "$hits" ]]; then
    echo "check_brand_boundary: hardcoded brand in $path"
    echo "$hits"
    fail=1
  fi
}

while IFS= read -r -d '' file; do
  scan_brand "$file"
done < <(find internal cmd -name '*.go' -print0 2> /dev/null || true)

pkg_hits="$(rg -n "$pattern_brand" pkg --glob '*.go' --glob '!pkg/branding/*' 2> /dev/null | rg -v 'ad-event-processor' || true)"
if [[ -n "$pkg_hits" ]]; then
  echo "check_brand_boundary: hardcoded brand outside pkg/branding:"
  echo "$pkg_hits"
  fail=1
fi

# Aggregate fail flag: exit 1 after all checks
if [[ "$fail" -ne 0 ]]; then
  echo "tier_a: FAILED"
  exit 1
fi

bash "$SCRIPTS/ci/prometheus_rules_check.sh"

echo "tier_a: OK"
