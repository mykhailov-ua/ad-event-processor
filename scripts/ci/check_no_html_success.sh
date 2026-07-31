#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

fail=0

check_dir() {
  local dir="$1"
  while IFS= read -r -d '' file; do
  case "$file" in
    *_test.go|*/testdata/*) continue ;;
  esac
  if rg -n 'Set\("Content-Type",\s*"text/html' "$file" >/dev/null 2>&1; then
    echo "check_no_html_success: text/html success Content-Type in $file"
    rg -n 'Set\("Content-Type",\s*"text/html' "$file" || true
    fail=1
  fi
  if rg -n 'HX-Request|hx-post|hx-get|WriteHTMX|HTMXError' "$file" >/dev/null 2>&1; then
    echo "check_no_html_success: HTMX reference in $file"
    rg -n 'HX-Request|hx-post|hx-get|WriteHTMX|HTMXError' "$file" || true
    fail=1
  fi
  done < <(find "$dir" -name '*.go' -print0)
}

check_dir internal/controlplane
check_dir internal/payment

if rg -n 'text/template' internal/controlplane internal/payment --glob '*.go' --glob '!*_test.go' >/dev/null 2>&1; then
  echo "check_no_html_success: text/template import in management/payment"
  rg -n 'text/template' internal/controlplane internal/payment --glob '*.go' --glob '!*_test.go' || true
  fail=1
fi

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

echo "check_no_html_success: OK"
