#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/lint_go_paths.sh"
cd "$ROOT"

mode="${1:-all}"

ensure_golangci_lint() {
	if [[ -z "$(which golangci-lint 2>/dev/null)" ]]; then
		echo "Installing golangci-lint..."
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6
	fi
	GOPATH="$(go env GOPATH)"
	if [[ -z "$GOPATH" ]]; then
		GOPATH="$HOME/go"
	fi
	export LINT_GOLANGCI="$GOPATH/bin/golangci-lint"
}

lint_extra_args() {
	local -n out=$1
	out=()
	# Incremental on PR/CI: contextcheck debt is retired file-by-file; new issues still fail.
	if [[ "${LINT_STRICT:-}" == "1" ]]; then
		return 0
	fi
	if [[ "${CI:-}" == "true" || "${LINT_NEW_FROM_REV:-}" == "1" ]]; then
		if git rev-parse --verify origin/main >/dev/null 2>&1; then
			out+=(--new-from-rev=origin/main)
		fi
	fi
}

verify_cold_excludes_hot_path() {
	local config=".golangci-cold.yaml"
	for dir in "${lint_go_hot_path_dirs[@]}"; do
		if rg -q "$dir/" "$config" 2>/dev/null; then
			echo "lint_go_gate: $config must not list hot-path exclude for $dir (cold run uses --skip-dirs)" >&2
			exit 1
		fi
	done
}

run_cold() {
	local extra=()
	lint_extra_args extra
	echo "lint_go_gate: cold path (errcheck, gocritic, noctx, bodyclose, contextcheck, errorlint, forbidigo)..."
	"$LINT_GOLANGCI" run -c .golangci-cold.yaml \
		--skip-dirs="$lint_go_hot_path_skip_re" \
		"${extra[@]}"
}

run_hot() {
	local extra=()
	lint_extra_args extra
	local pkgs=()
	for dir in "${lint_go_hot_path_dirs[@]}"; do
		[[ -d "$dir" ]] || continue
		pkgs+=("./${dir}/...")
	done
	if ((${#pkgs[@]} == 0)); then
		echo "lint_go_gate: hot path: no package directories present; skip" >&2
		return 0
	fi
	echo "lint_go_gate: hot path (govet, staticcheck, gocritic bug checks, errorlint, forbidigo panic/fatal)..."
	"$LINT_GOLANGCI" run -c .golangci-hot.yaml "${extra[@]}" "${pkgs[@]}"
}

ensure_golangci_lint

case "$mode" in
	cold)
		verify_cold_excludes_hot_path
		run_cold
		;;
	hot)
		run_hot
		bash "$SCRIPTS/ci/lint_go_hotpath_forbid_gate.sh"
		;;
	all)
		verify_cold_excludes_hot_path
		run_cold
		run_hot
		bash "$SCRIPTS/ci/lint_go_hotpath_forbid_gate.sh"
		;;
	*)
		echo "usage: $0 [cold|hot|all]" >&2
		exit 2
		;;
esac

echo "lint_go_gate: OK ($mode)"
