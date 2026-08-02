#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

fail=0

echo "tier_a: check docs layout..."
for forbidden in docs/MULTI_REGION.md docs/COMPLIANCE_MATRIX.md docs/MILESTONE.md docs/SELF_HOSTED.md docs/PROTECTION.md; do
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
	QUICKSTART.md
	RTB_PRODUCTION_RUNBOOK.md
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
	echo "docs layout: docs/runbooks/ must not exist (content in ARCHITECTURE.md / DEVELOPMENT.md)" >&2
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

echo "tier_a: check no milestone references..."
pattern_milestone='\bM[0-9]+'
scan_milestone() {
	local path="$1"
	if rg -n "$pattern_milestone" "$path" >/dev/null 2>&1; then
		echo "check_no_milestone_refs: milestone tag in $path"
		rg -n "$pattern_milestone" "$path" || true
		fail=1
	fi
}

while IFS= read -r -d '' file; do
	case "$file" in
		*/pb/*|*/sqlc/*|*_test.go|*/testdata/*) continue ;;
	esac
	case "$(basename "$file")" in
		*.pb.go|*_grpc.pb.go|*_vtproto.pb.go|*_bpfel.go|*_bpfeb.go) continue ;;
	esac
	scan_milestone "$file"
done < <(find internal cmd pkg tests -name '*.go' -print0 2>/dev/null || true)

while IFS= read -r -d '' file; do
	scan_milestone "$file"
done < <(find scripts -name '*.sh' -print0 2>/dev/null || true)

BASE="${1:-origin/main}"
if git rev-parse --verify "$BASE" >/dev/null 2>&1; then
	mapfile -t diff_hits < <(
		git diff "$BASE"...HEAD -- '*.go' '*.sh' \
			| rg "$pattern_milestone" \
			| rg -v 'milestoneTag|milestoneWord|check_no_milestone_refs' || true
	)
	if ((${#diff_hits[@]})); then
		echo "check_no_milestone_refs: forbidden milestone tag in diff:"
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

check_html_dir internal/controlplane
check_html_dir internal/payment

if rg -n 'text/template' internal/controlplane internal/payment --glob '*.go' --glob '!*_test.go' >/dev/null 2>&1; then
	echo "check_no_html_success: text/template import in management/payment"
	rg -n 'text/template' internal/controlplane internal/payment --glob '*.go' --glob '!*_test.go' || true
	fail=1
fi

echo "tier_a: check no slog in service files..."
mapfile -t slog_hits < <(
	rg -n 'slog\.' internal/controlplane/service_*.go \
		--glob '!internal/controlplane/service.go' 2>/dev/null || true
)

if ((${#slog_hits[@]})); then
	echo "check_no_service_slog: slog forbidden in service_*.go (except service.go lifecycle):"
	printf '  %s\n' "${slog_hits[@]}"
	fail=1
fi

echo "tier_a: check error handling in outbox/handlers..."
pattern_err='_ = (json\.Unmarshal|w\.Write)'
scan_err() {
	local path="$1"
	if rg -n "$pattern_err" "$path" >/dev/null 2>&1; then
		echo "check_error_handling: ignored error in $path"
		rg -n "$pattern_err" "$path" || true
		fail=1
	fi
}

while IFS= read -r -d '' file; do
	scan_err "$file"
done < <(find internal/controlplane -name 'outbox_*.go' ! -name '*_test.go' -print0 2>/dev/null || true)

while IFS= read -r -d '' file; do
	scan_err "$file"
done < <(find internal/controlplane -name 'handler_*.go' ! -name '*_test.go' -print0 2>/dev/null || true)

if [[ -d internal/controlplane/adminapi ]]; then
	while IFS= read -r -d '' file; do
		scan_err "$file"
	done < <(find internal/controlplane/adminapi -name '*_handlers.go' ! -name '*_test.go' -print0 2>/dev/null || true)
fi

echo "tier_a: check brand boundary..."
pattern_brand='BidShard|bidshard\.com'
scan_brand() {
	local path="$1"
	case "$path" in
		*/pb/*|*/sqlc/*|*_test.go|*/testdata/*) return 0 ;;
	esac
	case "$(basename "$path")" in
		*.pb.go|*_grpc.pb.go|*_vtproto.pb.go|*_bpfel.go|*_bpfeb.go) return 0 ;;
	esac
	if rg -n "$pattern_brand" "$path" >/dev/null 2>&1; then
		echo "check_brand_boundary: hardcoded brand in $path"
		rg -n "$pattern_brand" "$path" || true
		fail=1
	fi
}

while IFS= read -r -d '' file; do
	scan_brand "$file"
done < <(find internal cmd -name '*.go' -print0 2>/dev/null || true)

if rg -n "$pattern_brand" pkg --glob '*.go' --glob '!pkg/branding/*' >/dev/null 2>&1; then
	echo "check_brand_boundary: hardcoded brand outside pkg/branding:"
	rg -n "$pattern_brand" pkg --glob '*.go' --glob '!pkg/branding/*' || true
	fail=1
fi

if [[ "$fail" -ne 0 ]]; then
	echo "tier_a: FAILED"
	exit 1
fi

echo "tier_a: OK"
