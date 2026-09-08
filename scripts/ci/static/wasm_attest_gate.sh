#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
MAX_BYTES="${WASM_ATTEST_MAX_BYTES:-24576}"

bash "${ROOT}/scripts/build/wasm_attest.sh"

WASM="${WASM_ATTEST_OUT:-${ROOT}/var/wasm/attest.wasm}"
BYTES="$(wc -c < "${WASM}" | tr -d ' ')"
if [[ "${BYTES}" -gt "${MAX_BYTES}" ]]; then
  echo "wasm_attest_gate: ${BYTES} bytes exceeds cap ${MAX_BYTES}" >&2
  exit 1
fi

go test ./pkg/wasmattest/ -short -run WasmAttest -count=1

echo "wasm_attest_gate: ok bytes=${BYTES} cap=${MAX_BYTES}"
