#!/usr/bin/env bash
set -euo pipefail

# Role: PERIMETER_INTEL_DEFENSE.md structure gate (threat catalog + T2 runbook + checklist).
# Execution context: CI via pr_fast.
# Verify: bash scripts/ci/naming/perimeter_intel_doc.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

DOC="$ROOT/deploy/vendor/PERIMETER_INTEL_DEFENSE.md"

if [[ ! -f "$DOC" ]]; then
  echo "perimeter_intel_doc_gate: missing $DOC" >&2
  exit 1
fi

for heading in \
  '## Threat catalog (T1–T30)' \
  '## T2: Human-in-the-loop Sybil (operator runbook)' \
  '## Operator checklist (pre-production and quarterly)' \
  '## T12: Cross-session device reuse' \
  '## T15: TLS server persona'; do
  if ! rg -Fq "$heading" "$DOC"; then
    echo "perimeter_intel_doc_gate: missing section: $heading" >&2
    exit 1
  fi
done

for id in T1 T2 T12 T15 T21 T24 T28 T30; do
  if ! rg -q "\| ${id} \|" "$DOC"; then
    echo "perimeter_intel_doc_gate: threat catalog missing row for ${id}" >&2
    exit 1
  fi
done

bad_lines="$(rg -ni 'guaranteed block of all|blocks all scrapers|enterprise-grade' "$DOC" | rg -v 'must not|do not|No UI|not promise' || true)"
if [[ -n "$bad_lines" ]]; then
  echo "perimeter_intel_doc_gate: forbidden marketing prose in $DOC" >&2
  echo "$bad_lines" >&2
  exit 1
fi

if ! rg -q 'perimeter_intel_doc.sh' "$DOC"; then
  echo "perimeter_intel_doc_gate: operator checklist must cite perimeter_intel_doc.sh" >&2
  exit 1
fi

echo "perimeter_intel_doc_gate: ok"
