#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

fail=0
pattern='_ = (json\.Unmarshal|w\.Write)'

scan() {
	local path="$1"
	if rg -n "$pattern" "$path" >/dev/null 2>&1; then
		echo "check_error_handling: ignored error in $path"
		rg -n "$pattern" "$path" || true
		fail=1
	fi
}

while IFS= read -r -d '' file; do
	scan "$file"
done < <(find internal/controlplane -name 'outbox_*.go' ! -name '*_test.go' -print0 2>/dev/null || true)

while IFS= read -r -d '' file; do
	scan "$file"
done < <(find internal/controlplane -name 'handler_*.go' ! -name '*_test.go' -print0 2>/dev/null || true)

while IFS= read -r -d '' file; do
	scan "$file"
done < <(find internal/controlplane/adminapi -name '*_handlers.go' ! -name '*_test.go' -print0 2>/dev/null || true)

if [[ "$fail" -ne 0 ]]; then
	exit 1
fi

echo "check_error_handling: OK"
