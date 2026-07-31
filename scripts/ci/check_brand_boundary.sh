#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

fail=0
pattern='BidShard|bidshard\.com'

scan_go() {
	local path="$1"
	case "$path" in
		*/pb/*|*/sqlc/*|*_test.go|*/testdata/*) return 0 ;;
	esac
	case "$(basename "$path")" in
		*.pb.go|*_grpc.pb.go|*_vtproto.pb.go|*_bpfel.go|*_bpfeb.go) return 0
	esac
	if rg -n "$pattern" "$path" >/dev/null 2>&1; then
		echo "check_brand_boundary: hardcoded brand in $path"
		rg -n "$pattern" "$path" || true
		fail=1
	fi
}

while IFS= read -r -d '' file; do
	scan_go "$file"
done < <(find internal cmd -name '*.go' -print0 2>/dev/null || true)

if rg -n "$pattern" pkg --glob '*.go' --glob '!pkg/branding/*' >/dev/null 2>&1; then
	echo "check_brand_boundary: hardcoded brand outside pkg/branding:"
	rg -n "$pattern" pkg --glob '*.go' --glob '!pkg/branding/*' || true
	fail=1
fi

if [[ "$fail" -ne 0 ]]; then
	exit 1
fi

echo "check_brand_boundary: OK"
