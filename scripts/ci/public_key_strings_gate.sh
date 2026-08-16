#!/usr/bin/env bash
# V2-D.D3: release tracker must not embed plaintext production Ed25519 pubkey hex.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

BIN="${1:-}"
if [[ -z "$BIN" ]]; then
	OUT="${TMPDIR:-/tmp}/public-key-strings-gate-$$"
	mkdir -p "$OUT"
	trap 'rm -rf "$OUT"' EXIT
	BIN="$OUT/tracker"
	echo "public_key_strings_gate: building tracker probe"
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$BIN" ./cmd/tracker
fi

if [[ ! -f "$BIN" ]]; then
	echo "public_key_strings_gate: missing binary: $BIN" >&2
	exit 1
fi

FORBIDDEN=(
	'BEGIN PUBLIC'
	'ede21d8e759af2ba68a74149d28f37a859d33497accee01e8f8ac712bd455c70'
)

for pat in "${FORBIDDEN[@]}"; do
	if strings "$BIN" | rg -qi "$pat"; then
		echo "public_key_strings_gate: forbidden pattern '$pat' in $BIN" >&2
		exit 1
	fi
done

echo "public_key_strings_gate: OK ($BIN)"
