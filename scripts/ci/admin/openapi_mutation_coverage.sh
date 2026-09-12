#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

if [[ ! -d web/src/domains ]]; then
  echo "openapi_mutation_coverage: skipped (web/src absent)"
  exit 0
fi

missing=0
while IFS= read -r file; do
  case "$file" in
    */api/* | */lib/* | */shell/* | *.test.* | *.spec.*) continue ;;
  esac
  if rg -q "fetch\\(|api\\." "$file" 2> /dev/null; then
    if ! rg -q "openapi|/api/v1/" "$file" 2> /dev/null; then
      echo "openapi_mutation_coverage: possible uncovered mutation: $file"
      missing=$((missing + 1))
    fi
  fi
done < <(find web/src/domains -type f \( -name '*.ts' -o -name '*.tsx' \) | sort)

if [[ "$missing" -gt 0 ]]; then
  echo "openapi_mutation_coverage: $missing files flagged (review checklist)"
fi

if [[ ! -f docs/INTEGRATIONS.md ]] || ! rg -q '/api/v1/openapi.yaml' docs/INTEGRATIONS.md; then
  echo "openapi_mutation_coverage: docs/INTEGRATIONS.md missing OpenAPI link"
  exit 1
fi

echo "openapi_mutation_coverage: ok"
