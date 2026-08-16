#!/usr/bin/env bash
# M2.6: Go HKDF/MCK derivation vs OpenSSL 3 kdf (requires openssl in PATH).
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if ! command -v openssl >/dev/null 2>&1; then
	echo "license_differential_gate: skip (openssl not in PATH)"
	exit 0
fi

if ! openssl kdf -help >/dev/null 2>&1; then
	echo "license_differential_gate: skip (openssl kdf subcommand unavailable)"
	exit 0
fi

go test -tags=differential ./internal/licensing/ \
	-run 'HKDF_DifferentialOpenSSL|DeriveMCK_DifferentialOpenSSL' \
	-count=1
echo "license_differential_gate: OK"
