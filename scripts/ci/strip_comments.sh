#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

write=0
if [[ "${1:-}" == "--write" ]]; then
	write=1
	shift
fi

args=()
if [[ "$write" -eq 1 ]]; then
	args+=(--write)
fi

exec go run ./cmd/strip-comments "${args[@]}"
