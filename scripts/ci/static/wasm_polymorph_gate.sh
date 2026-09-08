#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
TMP="${ROOT}/var/ci/polymorph_gate"
rm -rf "${TMP}"
mkdir -p "${TMP}"

build_seed() {
  local seed="$1"
  local out="${TMP}/attest-${seed}.wasm"
  WASM_ATTEST_SEED="${seed}" \
    WASM_ATTEST_SKIP_EMBED=1 \
    WASM_ATTEST_OUT="${out}" \
    bash "${ROOT}/scripts/build/wasm_attest.sh" --seed="${seed}" >&2
  printf '%s' "${out}"
}

DEFAULT="$(build_seed 0)"
A="$(build_seed tenant-a)"
B="$(build_seed tenant-b)"

DEFAULT_SHA="$(sha256sum "${DEFAULT}" | awk '{print $1}')"
A_SHA="$(sha256sum "${A}" | awk '{print $1}')"
B_SHA="$(sha256sum "${B}" | awk '{print $1}')"

if [[ "${DEFAULT_SHA}" == "${A_SHA}" || "${A_SHA}" == "${B_SHA}" ]]; then
  echo "wasm_polymorph_gate: expected distinct sha256 per seed" >&2
  echo " default=${DEFAULT_SHA} a=${A_SHA} b=${B_SHA}" >&2
  exit 1
fi

BYTES_A="$(wc -c < "${A}" | tr -d ' ')"
MAX_BYTES="${WASM_ATTEST_MAX_BYTES:-24576}"
if [[ "${BYTES_A}" -gt "${MAX_BYTES}" ]]; then
  echo "wasm_polymorph_gate: ${BYTES_A} bytes exceeds cap ${MAX_BYTES}" >&2
  exit 1
fi

bash "${ROOT}/scripts/ci/static/wasm_attest_gate.sh"

export WASM_POLYMORPH_GATE=1
export WASM_POLYMORPH_PATHS="${DEFAULT}:${A}:${B}"
go test ./pkg/wasmattest/ -short -run WasmAttest_polymorphVariants -count=1

echo "wasm_polymorph_gate: ok seeds=0,tenant-a,tenant-b bytes=${BYTES_A}"
