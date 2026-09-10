#!/usr/bin/env bash
# Role: Build gzip+base64 obfuscated remote installer stub for marketing CDN.
# Execution context: Operator machine before deploy-marketing; reads scripts/install/get.sh.
# Env: MARKETING_DOMAIN (default bidshard.com); OBFUSCATE_GET_OUT (default deploy/marketing/get.sh).
# Verify: MARKETING_DOMAIN=bidshard.com bash scripts/install/obfuscate_get.sh && bash deploy/marketing/get.sh 2>&1 | head -1
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

SRC="$ROOT/scripts/install/get.sh"
DOMAIN="${MARKETING_DOMAIN:-bidshard.com}"
OUT="${OBFUSCATE_GET_OUT:-$ROOT/deploy/marketing/get.sh}"
RELEASE_BASE="https://${DOMAIN}/releases"
GET_URL="https://${DOMAIN}/get.sh"

if [[ ! -f "$SRC" ]]; then
  echo "obfuscate_get: missing source $SRC" >&2
  exit 1
fi

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

{
  printf '%s\n' '#!/usr/bin/env bash'
  printf '%s\n' "export AD_EVENT_PROCESSOR_GET_SCRIPT_URL=\"${GET_URL}\""
  printf '%s\n' "export AD_EVENT_PROCESSOR_RELEASE_BASE_URL=\"${RELEASE_BASE}\""
  tail -n +2 "$SRC"
} > "$tmp"

payload="$(gzip -c -9 "$tmp" | base64 -w0 2>/dev/null || gzip -c -9 "$tmp" | base64)"

mkdir -p "$(dirname "$OUT")"
{
  cat << 'HDR'
#!/usr/bin/env bash
set -euo pipefail
__aed_need(){ command -v "$1" >/dev/null 2>&1 || { printf 'bidshard-bootstrap: need %s\n' "$1" >&2; exit 127; }; }
__aed_need bash
__aed_need base64
__aed_need gzip
__aed_run(){
  local _p="$1"
  printf '%s' "$_p" | base64 -d | gzip -dc
}
HDR
  printf '__aed_blob="%s"\n' "$payload"
  cat << 'TAIL'
eval "$(__aed_run "$__aed_blob")" "$@"
TAIL
} > "$OUT"

chmod 755 "$OUT"
printf 'obfuscate_get: wrote %s (%s bytes) for %s\n' "$OUT" "$(wc -c < "$OUT" | tr -d ' ')" "$GET_URL"
