#!/usr/bin/env bash
# Role: One-shot dev appliance bootstrap: codegen, demo license, GeoIP, compose, admin seed.
# Execution context: Fresh clone on dev host; wraps stack.sh and seed_admin.sh.
# Env knobs: --profile ingest-only|full; --with-bpf; --skip-gen, --skip-geoip, --skip-up, --skip-seed.
# Verify: bash scripts/install/appliance_bootstrap.sh --dry-run
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/installer_env.sh"
source "$SCRIPTS/lib/go.sh"
cd "$ROOT"

PROFILE="ingest-only"
SKIP_GEN=0
SKIP_BPF=1
SKIP_GEOIP=0
SKIP_UP=0
SKIP_SEED=0
DRY_RUN=0

usage() {
  echo "usage: $0 [--profile ingest-only|full] [--with-bpf] [--skip-gen] [--skip-geoip] [--skip-up] [--skip-seed] [--dry-run]" >&2
  echo "  One-shot dev appliance bootstrap: deps, codegen, demo license, GeoIP, compose, admin seed." >&2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --profile)
      PROFILE="${2:-}"
      shift 2
      ;;
    --with-bpf)
      SKIP_BPF=0
      shift
      ;;
    --skip-gen)
      SKIP_GEN=1
      shift
      ;;
    --skip-geoip)
      SKIP_GEOIP=1
      shift
      ;;
    --skip-up)
      SKIP_UP=1
      shift
      ;;
    --skip-seed)
      SKIP_SEED=1
      shift
      ;;
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "appliance-bootstrap: unknown argument: $1" >&2
      usage
      exit 2
      ;;
  esac
done

case "$PROFILE" in
  ingest-only | full) ;;
  *)
    echo "appliance-bootstrap: invalid profile: $PROFILE (use ingest-only or full)" >&2
    exit 2
    ;;
esac

log() { printf 'appliance-bootstrap: %s\n' "$*"; }
warn() { printf 'appliance-bootstrap: WARN %s\n' "$*" >&2; }
die() {
  printf 'appliance-bootstrap: ERROR %s\n' "$*" >&2
  exit 1
}

read_env_var() {
  local key="$1"
  if [[ -f .env ]]; then
    grep -m1 "^${key}=" .env 2> /dev/null | cut -d= -f2- || true
  else
    echo ""
  fi
}

set_env_key() {
  local key="$1"
  local val="$2"
  if grep -q "^${key}=" .env 2> /dev/null; then
    sed -i "s|^${key}=.*|${key}=${val}|" .env
  else
    echo "${key}=${val}" >> .env
  fi
}

check_deps() {
  log "checking host dependencies"
  local missing=0
  for cmd in docker curl make; do
    if ! command -v "$cmd" > /dev/null 2>&1; then
      warn "missing command: $cmd"
      missing=1
    fi
  done
  if ! aed_go_bin > /dev/null 2>&1; then
    warn "go not found (set AD_EVENT_PROCESSOR_GO_BIN)"
    missing=1
  fi
  if ! docker compose version > /dev/null 2>&1; then
    warn "docker compose v2 not available"
    missing=1
  fi
  if ! docker info > /dev/null 2>&1; then
    warn "docker daemon not reachable"
    missing=1
  fi
  if [[ "$missing" -ne 0 ]]; then
    die "dependency check failed"
  fi
  log "dependencies ok"
}

ensure_env() {
  log "ensuring .env"
  if [[ ! -f .env ]]; then
    cp .env.example .env
  fi
  mkdir -p var deploy/geoip
  if ! grep -q '^INSTALL_BOOTSTRAP_TOKEN=' .env; then
    echo "INSTALL_BOOTSTRAP_TOKEN=$(openssl rand -hex 32)" >> .env
  elif [[ -z "$(read_env_var INSTALL_BOOTSTRAP_TOKEN)" ]]; then
    set_env_key INSTALL_BOOTSTRAP_TOKEN "$(openssl rand -hex 32)"
  fi
  local pub_file="deploy/vendor/license_public.key"
  if [[ -f "$pub_file" ]] && [[ -z "$(read_env_var AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY)" ]]; then
    set_env_key AD_EVENT_PROCESSOR_LICENSE_PUBLIC_KEY "$(tr -d '[:space:]' < "$pub_file")"
  fi
  set_env_key AD_EVENT_PROCESSOR_LICENSE_MODE file
  set_env_key AD_EVENT_PROCESSOR_LICENSE_PATH var/license.jwt
}

run_codegen() {
  if [[ "$SKIP_GEN" -eq 1 ]]; then
    log "skipping codegen (--skip-gen)"
    return 0
  fi
  log "running make gen"
  make gen
  if [[ "$SKIP_BPF" -eq 0 ]]; then
    log "running make gen bpf-dev"
    if ! make gen bpf-dev; then
      warn "bpf-dev build failed (optional for ingest-only dev)"
    fi
  else
    log "skipping bpf-dev (pass --with-bpf to build edge probes)"
  fi
}

issue_demo_license() {
  local dest="var/license.jwt"
  if [[ -s "$dest" ]]; then
    log "license jwt already present at ${dest}"
    return 0
  fi
  local priv="deploy/vendor/license_private.key"
  if [[ ! -f "$priv" ]]; then
    warn "vendor private key missing; issue pilot JWT manually:"
    warn "  VENDOR_TRIAL_FORCE=1 go run ./cmd/license-issue --sku pilot --customer dev-local --force --force-reason appliance_bootstrap --out var/license.jwt"
    return 0
  fi
  log "issuing pilot demo license"
  VENDOR_TRIAL_FORCE=1 aed_go_run ./cmd/license-issue \
    --sku pilot \
    --customer dev-local \
    --deployment-id dev-appliance-bootstrap \
    --force \
    --force-reason appliance_bootstrap \
    --out "$dest"
}

fetch_geoip() {
  if [[ "$SKIP_GEOIP" -eq 1 ]]; then
    log "skipping geoip fetch (--skip-geoip)"
    return 0
  fi
  local dest="deploy/geoip/GeoLite2-Country.mmdb"
  if [[ -f "$dest" && -s "$dest" ]]; then
    log "geoip database present at ${dest}"
    return 0
  fi
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
  if [[ -z "${MAXMIND_LICENSE_KEY:-}" ]]; then
    warn "MAXMIND_LICENSE_KEY unset; geo filters use dev fallbacks until database is installed"
    return 0
  fi
  local edition="${MAXMIND_EDITION_ID:-GeoLite2-Country}"
  log "downloading MaxMind ${edition}"
  python3 - "$edition" "$dest" << 'PY'
import os, sys, tarfile, tempfile, urllib.request, shutil

edition, dest = sys.argv[1], sys.argv[2]
key = os.environ.get("MAXMIND_LICENSE_KEY", "").strip()
if not key:
    raise SystemExit("MAXMIND_LICENSE_KEY missing")
url = (
    "https://download.maxmind.com/app/geoip_download"
    f"?edition_id={edition}&license_key={key}&suffix=tar.gz"
)
os.makedirs(os.path.dirname(dest) or ".", exist_ok=True)
with tempfile.NamedTemporaryFile(suffix=".tar.gz", delete=False) as tmp:
    tmp_path = tmp.name
try:
    with urllib.request.urlopen(url, timeout=300) as resp:
        if resp.status != 200:
            body = resp.read(1024).decode("utf-8", "replace")
            raise SystemExit(f"maxmind download failed: HTTP {resp.status}: {body}")
        shutil.copyfileobj(resp, tmp)
    extracted = False
    with tarfile.open(tmp_path, "r:gz") as archive:
        for member in archive.getmembers():
            if not member.isfile() or not member.name.endswith(".mmdb"):
                continue
            member_file = archive.extractfile(member)
            if member_file is None:
                continue
            staging = dest + ".staging"
            with open(staging, "wb") as out:
                shutil.copyfileobj(member_file, out)
            os.replace(staging, dest)
            extracted = True
            break
    if not extracted:
        raise SystemExit("no .mmdb file in maxmind archive")
finally:
    try:
        os.remove(tmp_path)
    except OSError:
        pass
print(f"geoip installed: {dest}")
PY
}

wait_http_health() {
  local name="$1"
  local url="$2"
  local attempts="${3:-90}"
  local i=0
  while [[ $i -lt $attempts ]]; do
    if curl -sf "$url" > /dev/null 2>&1; then
      log "${name} healthy at ${url}"
      return 0
    fi
    sleep 2
    i=$((i + 1))
  done
  die "${name} health check failed: ${url}"
}

wait_stack_health() {
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
  local mgmt="${MANAGEMENT_PORT:-8188}"
  local tracker="${SERVER_PORT:-8181}"
  local processor="${PROCESSOR_PORT:-8186}"
  wait_http_health control "http://127.0.0.1:${mgmt}/health"
  wait_http_health tracker "http://127.0.0.1:${tracker}/health"
  wait_http_health processor "http://127.0.0.1:${processor}/health"
}

start_stack() {
  if [[ "$SKIP_UP" -eq 1 ]]; then
    log "skipping compose up (--skip-up)"
    return 0
  fi
  log "building compose images"
  bash "$SCRIPTS/dev/stack/stack.sh" build
  log "starting profile ${PROFILE}"
  case "$PROFILE" in
    full)
      bash "$SCRIPTS/dev/stack/stack.sh" full
      ;;
    ingest-only)
      bash "$SCRIPTS/dev/stack/stack.sh" ingest-only
      ;;
  esac
  wait_stack_health
}

seed_admin() {
  if [[ "$SKIP_SEED" -eq 1 ]]; then
    log "skipping admin seed (--skip-seed)"
    return 0
  fi
  log "seeding platform admin"
  bash "$SCRIPTS/dev/stack/seed_admin.sh"
}

print_summary() {
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
  local mgmt="${MANAGEMENT_PORT:-8188}"
  local tracker="${SERVER_PORT:-8181}"
  local email="${ADMIN_BOOTSTRAP_EMAIL:-admin@test.local}"
  local password="${ADMIN_BOOTSTRAP_PASSWORD:-Password123!}"
  local api_key="${ADMIN_API_KEY:-dev-admin-api-key-change-me}"
  local control_url="http://127.0.0.1:${mgmt}"
  local click_url="http://127.0.0.1:${tracker}/click?campaign_id={campaign_id}&sub1={sub1}"

  cat << EOF

appliance-bootstrap: ready (profile=${PROFILE})

Control UI:  ${control_url}
Login:       ${control_url}/login
Email:       ${email}
Password:    ${password}

Click URL:   ${click_url}
Track URL:   http://127.0.0.1:${tracker}/track?campaign_id={campaign_id}&type=impression

Import bundled integration templates:
  curl -sS -X POST ${control_url}/api/v1/integration/templates/import \\
    -H "Content-Type: application/json" \\
    -H "X-Admin-API-Key: ${api_key}" \\
    -d '{}'

Smoke:       bash scripts/dev/stack/smoke_local.sh
Status:      bash scripts/dev/stack/stack.sh status

Pilot license: var/license.jwt (re-issue: go run ./cmd/license-issue --sku pilot --customer dev-local)

EOF
}

main() {
  check_deps
  ensure_env
  run_codegen
  issue_demo_license
  fetch_geoip
  if [[ "$DRY_RUN" -eq 1 ]]; then
    log "dry-run complete (codegen/license/geoip only; stack not started)"
    print_summary
    return 0
  fi
  start_stack
  seed_admin
  if ! bash "$SCRIPTS/dev/stack/smoke_local.sh"; then
    die "post-bootstrap smoke failed"
  fi
  print_summary
}

main "$@"
