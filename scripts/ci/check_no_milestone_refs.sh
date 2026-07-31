#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

fail=0
pattern='\bM[0-9]+'

scan_file() {
	local path="$1"
	if rg -n "$pattern" "$path" >/dev/null 2>&1; then
		echo "check_no_milestone_refs: milestone tag in $path"
		rg -n "$pattern" "$path" || true
		fail=1
	fi
}

scan_go() {
	local path="$1"
	case "$path" in
		*/pb/*|*/sqlc/*|*_test.go|*/testdata/*|cmd/check-comments/*) return 0 ;;
	esac
	case "$(basename "$path")" in
		*.pb.go|*_grpc.pb.go|*_vtproto.pb.go|*_bpfel.go|*_bpfeb.go) return 0
	esac
	scan_file "$path"
}

scan_sh() {
	local path="$1"
	scan_file "$path"
}

while IFS= read -r -d '' file; do
	scan_go "$file"
done < <(find internal cmd pkg tests -name '*.go' -print0 2>/dev/null || true)

while IFS= read -r -d '' file; do
	scan_sh "$file"
done < <(find scripts -name '*.sh' -print0 2>/dev/null || true)

if [[ -f .cursor/COMPLIANCE_MATRIX.md ]]; then
	scan_file .cursor/COMPLIANCE_MATRIX.md
fi

BASE="${1:-origin/main}"
if git rev-parse --verify "$BASE" >/dev/null 2>&1; then
	mapfile -t diff_hits < <(
		git diff "$BASE"...HEAD -- '*.go' '*.sh' \
			| rg "$pattern" \
			| rg -v 'milestoneTag|milestoneWord|check_no_milestone_refs' || true
	)
	if ((${#diff_hits[@]})); then
		echo "check_no_milestone_refs: forbidden milestone tag in diff:"
		printf '  %s\n' "${diff_hits[@]}"
		fail=1
	fi
fi

if [[ "$fail" -ne 0 ]]; then
	exit 1
fi

echo "check_no_milestone_refs: OK"
