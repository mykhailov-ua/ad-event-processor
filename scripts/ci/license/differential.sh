#!/usr/bin/env bash

set -euo pipefail

# Role: License gate: OpenSSL KDF differential vs Go HKDF.
# Execution context: CI license-verify tier or release QA.
# Invariants/contracts enforced: Required rows fail closed; optional rows use skip_gate with env flags.
# Verify: bash scripts/ci/license/differential.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

if ! command -v openssl > /dev/null 2>&1; then
  echo "license_differential_gate: skip (openssl not in PATH)"
  exit 0
fi

if ! openssl kdf -help > /dev/null 2>&1; then
  echo "license_differential_gate: skip (openssl kdf subcommand unavailable)"
  exit 0
fi

go test -tags=differential ./internal/licensing/ \
  -run 'HKDF_DifferentialOpenSSL|DeriveMCK_DifferentialOpenSSL' \
  -count=1
echo "license_differential_gate: OK"
