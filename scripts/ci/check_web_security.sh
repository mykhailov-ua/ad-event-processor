#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if rg -n '\bconsole\.(log|debug|info)\(' web/src --glob '*.js' 2>/dev/null; then
  echo "Error: console.log/debug/info in web/src (use dev guards or remove)."
  exit 1
fi

if rg -n 'from ['\''\"]react|\.jsx$' web/src 2>/dev/null; then
  echo "Error: React or JSX references in web/src."
  exit 1
fi

if rg -n 'password=|install_token=|secret_key=' web/src/views --glob '*.js' -i 2>/dev/null; then
  echo "Error: possible secret in URL query in views."
  exit 1
fi

echo "Web security literal check: OK"
