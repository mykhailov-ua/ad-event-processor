#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if rg -n '\bconsole\.(log|debug|info)\(' web/src --glob '*.js' 2> /dev/null; then
  echo "Error: console.log/debug/info in web/src (use dev guards or remove)."
  exit 1
fi

if rg -l '\.jsx$' web/src 2> /dev/null | grep -q .; then
  echo "Error: .jsx files not allowed in web/src (use .tsx with jsx: react-jsx)."
  exit 1
fi

# React imports allowed under pages/, components/, helpers/use_*.ts, and entry shells.
react_violations=()
while IFS= read -r file; do
  case "$file" in
    web/src/pages/* | */web/src/pages/* | web/src/components/* | */web/src/components/* | web/src/helpers/use_*.ts | */web/src/helpers/use_*.ts | web/src/main.tsx | */web/src/main.tsx | web/src/login.tsx | */web/src/login.tsx | web/src/app_*.tsx | */web/src/app_*.tsx | web/src/login_boot.tsx | */web/src/login_boot.tsx | web/src/main_mount.tsx | */web/src/main_mount.tsx | web/src/standalone_*.tsx | */web/src/standalone_*.tsx) ;;
    *) react_violations+=("$file") ;;
  esac
done < <(rg -l "from ['\"]react" web/src 2> /dev/null || true)

if [ "${#react_violations[@]}" -gt 0 ]; then
  echo "Error: React imports outside pages/components/helpers/use_* or entry shells:"
  printf '  %s\n' "${react_violations[@]}"
  exit 1
fi

if rg -n 'password=|install_token=|secret_key=' web/src/views --glob '*.js' -i 2> /dev/null; then
  echo "Error: possible secret in URL query in views."
  exit 1
fi

echo "Web security literal check: OK"
