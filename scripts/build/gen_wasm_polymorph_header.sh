#!/usr/bin/env bash
# Generate wasm/attest/polymorph_seed.h from WASM_ATTEST_SEED (install-time polymorph knobs).
# Verify: WASM_ATTEST_SEED=deadbeef bash scripts/build/gen_wasm_polymorph_header.sh /tmp/polymorph_seed.h
set -euo pipefail

OUT="${1:-}"
if [[ -z "${OUT}" ]]; then
  echo "usage: $0 <output.h>" >&2
  exit 2
fi

SEED_RAW="${WASM_ATTEST_SEED:-0}"
SEED_HEX="$(printf '%s' "${SEED_RAW}" | sha256sum | awk '{print $1}')"
SEED_U32="0x${SEED_HEX:0:8}"
JUNK_LEN=$((16 + (16#${SEED_HEX:8:2}) % 48))
EXTRA_FN=$((16#${SEED_HEX:10:2} % 4))

mkdir -p "$(dirname "${OUT}")"

{
  echo "/* generated; do not edit */"
  echo "#ifndef AAD_POLY_SEED_H"
  echo "#define AAD_POLY_SEED_H"
  echo "#include <stdint.h>"
  echo "#define AAD_POLY_SEED ${SEED_U32}u"
  echo "#define AAD_POLY_JUNK_LEN ${JUNK_LEN}u"
  echo "#define AAD_POLY_EXTRA_FN ${EXTRA_FN}u"
  echo "static const uint8_t aad_poly_junk_impl[${JUNK_LEN}] = {"
  for ((i = 0; i < JUNK_LEN; i++)); do
    byte=$((16#${SEED_HEX:$((2 + (i % 30))):2}))
    if ((i > 0)); then
      printf ','
    fi
    if ((i % 12 == 0)); then
      printf '\n '
    fi
    printf ' 0x%02x' "${byte}"
  done
  echo
  echo "};"
  echo "#undef aad_poly_junk"
  echo "#define aad_poly_junk aad_poly_junk_impl"
  echo "#endif"
} > "${OUT}"
