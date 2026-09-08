#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
OUT="${WASM_ATTEST_OUT:-${ROOT}/var/wasm/attest.wasm}"
SRC_DIR="${ROOT}/wasm/attest"
MANIFEST="${ROOT}/var/wasm/attest.manifest"
WASI_SDK_DIR="${WASI_SDK_DIR:-${ROOT}/var/wasi-sdk}"
POLY_HEADER="${WASM_ATTEST_POLY_HEADER:-${ROOT}/var/wasm/polymorph_seed.h}"
SEED="${WASM_ATTEST_SEED:-0}"
SKIP_EMBED="${WASM_ATTEST_SKIP_EMBED:-0}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --seed=*)
      SEED="${1#*=}"
      shift
      ;;
    --seed)
      SEED="${2:-0}"
      shift 2
      ;;
    --out=*)
      OUT="${1#*=}"
      shift
      ;;
    *)
      shift
      ;;
  esac
done

export WASM_ATTEST_SEED="${SEED}"
bash "${ROOT}/scripts/build/gen_wasm_polymorph_header.sh" "${POLY_HEADER}"

ensure_wasi_sdk() {
  if [[ -n "${CLANG:-}" && -n "${WASM_LD:-}" ]]; then
    return
  fi
  if [[ -x "${WASI_SDK_DIR}/bin/clang" && -x "${WASI_SDK_DIR}/bin/wasm-ld" ]]; then
    export CLANG="${WASI_SDK_DIR}/bin/clang"
    export WASM_LD="${WASI_SDK_DIR}/bin/wasm-ld"
    return
  fi
  if [[ "${WASM_ATTEST_FETCH_WASI_SDK:-0}" != "1" ]]; then
    cat >&2 << EOF
wasm_attest: need wasm-ld. Options:
  1) apt install clang lld (llvm wasm linker)
  2) export CLANG and WASM_LD to a wasm-capable toolchain
  3) WASM_ATTEST_FETCH_WASI_SDK=1 bash scripts/build/wasm_attest.sh
EOF
    exit 1
  fi
  local ver="24.0"
  local arch="x86_64"
  local tarball="wasi-sdk-${ver}-${arch}-linux.tar.gz"
  local extracted="wasi-sdk-${ver}-${arch}-linux"
  local url="https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-${ver%%.*}/${tarball}"
  mkdir -p "${ROOT}/var"
  if [[ ! -f "${ROOT}/var/${tarball}" ]]; then
    curl -fsSL "${url}" -o "${ROOT}/var/${tarball}"
  fi
  rm -rf "${WASI_SDK_DIR}"
  tar -xzf "${ROOT}/var/${tarball}" -C "${ROOT}/var"
  mv "${ROOT}/var/${extracted}" "${WASI_SDK_DIR}"
  export CLANG="${WASI_SDK_DIR}/bin/clang"
  export WASM_LD="${WASI_SDK_DIR}/bin/wasm-ld"
}

mkdir -p "$(dirname "$OUT")"
ensure_wasi_sdk

CC_BIN="${CLANG:-clang}"
LD_BIN="${WASM_LD:-wasm-ld}"

if ! command -v "${CC_BIN}" > /dev/null 2>&1; then
  echo "wasm_attest: compiler not found: ${CC_BIN}" >&2
  exit 1
fi

"${CC_BIN}" --target=wasm32-unknown-unknown \
  -Oz \
  -ffunction-sections \
  -fdata-sections \
  -nostdlib \
  -fno-builtin \
  -I"${SRC_DIR}" \
  -include "${POLY_HEADER}" \
  -fuse-ld="${LD_BIN}" \
  -Wl,--no-entry \
  -Wl,--strip-all \
  -Wl,--initial-memory=131072 \
  -Wl,--max-memory=131072 \
  -Wl,--export=aad_abi_version \
  -Wl,--export=aad_data_off \
  -Wl,--export=aad_pow_solve \
  -Wl,--export=aad_float_noise_ieee \
  -Wl,--export=aad_sha256_one_shot \
  -Wl,--export=aad_bench_mul \
  -Wl,--export=aad_poly_seed \
  -Wl,--export=aad_poly_tag \
  -o "${OUT}" \
  "${SRC_DIR}/attest.c" \
  "${SRC_DIR}/polymorph.c" \
  "${SRC_DIR}/sha256.c"

if command -v wasm-opt > /dev/null 2>&1; then
  wasm-opt -Oz "${OUT}" -o "${OUT}"
fi

if [[ "${SKIP_EMBED}" != "1" ]]; then
  EMBED_DST="${ROOT}/internal/track/attest.wasm"
  cp "${OUT}" "${EMBED_DST}"
fi

BYTES="$(wc -c < "${OUT}" | tr -d ' ')"
SHA="$(sha256sum "${OUT}" | awk '{print $1}')"
{
  echo "bytes=${BYTES}"
  echo "sha256=${SHA}"
  echo "abi_version=1"
  echo "seed=${SEED}"
  echo "exports=memory,aad_abi_version,aad_pow_solve,aad_float_noise_ieee,aad_sha256_one_shot,aad_bench_mul,aad_poly_seed,aad_poly_tag"
} > "${MANIFEST}"

echo "wasm_attest: wrote ${OUT} (${BYTES} bytes) seed=${SEED} sha256=${SHA}"
