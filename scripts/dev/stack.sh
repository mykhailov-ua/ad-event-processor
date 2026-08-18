#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/installer_env.sh"
cd "$ROOT"

espx_read_env() {
  installer_read_env "$1"
}

espx_use_release_images() {
  installer_use_release_images
}

espx_ingress_enabled() {
  [[ "$(espx_read_env INGRESS_ENABLED)" == "1" ]]
}

espx_cpu_isolation_enabled() {
  [[ "$(espx_read_env CPU_ISOLATION_ENABLED)" == "1" ]]
}

espx_sysctl_auto_apply_enabled() {
  [[ "$(espx_read_env EDGE_SYSCTL_AUTO_APPLY)" == "1" ]]
}

espx_stack_hardening() {
  if [[ ! -x "$SCRIPTS/ops/sysctl.sh" ]]; then
    return 0
  fi
  if espx_sysctl_auto_apply_enabled; then
    if [[ "$(id -u)" -eq 0 ]]; then
      echo "stack.sh: applying host sysctl (EDGE_SYSCTL_AUTO_APPLY=1)" >&2
      bash "$SCRIPTS/ops/sysctl.sh" apply || echo "stack.sh: WARN sysctl apply failed" >&2
    else
      if ! bash "$SCRIPTS/ops/sysctl.sh" verify 2> /dev/null; then
        echo "stack.sh: WARN sysctl not applied — run: sudo bash scripts/ops/sysctl.sh apply" >&2
        echo "stack.sh: WARN recreate listeners after somaxconn change (see docs/EDGE_CASES.md)" >&2
      fi
    fi
  else
    bash "$SCRIPTS/ops/sysctl.sh" verify 2> /dev/null \
      || echo "stack.sh: hint: EDGE_SYSCTL_AUTO_APPLY=1 or sudo bash scripts/ops/sysctl.sh apply" >&2
  fi

  if espx_cpu_isolation_enabled && command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then
    if ! bash "$SCRIPTS/ops/cpu_isolation.sh" verify 2> /dev/null; then
      echo "stack.sh: WARN cpu isolation verify failed (tracker-0 running with profile cpu-isolation?)" >&2
    fi
  fi
}

espx_compose() {
  local -a env_args=()
  local -a file_args=(-f "$ROOT/docker-compose.yaml")
  local -a profile_args=()
  if espx_cpu_isolation_enabled; then
    file_args+=(-f "$ROOT/deploy/compose/docker-compose.cpu-isolation.yaml")
    profile_args+=(--profile cpu-isolation)
  fi
  if espx_use_release_images; then
    file_args+=(-f "$ROOT/deploy/compose/docker-compose.release.yaml")
    if [[ -z "${AD_EVENT_PROCESSOR_APP_IMAGE:-}" ]]; then
      AD_EVENT_PROCESSOR_APP_IMAGE="$(installer_release_app_image)"
      export AD_EVENT_PROCESSOR_APP_IMAGE
    fi
  fi
  if [[ -f "$ROOT/.env" ]]; then
    env_args+=(--env-file "$ROOT/.env")
  fi
  if [[ -f "$ROOT/install.compose.env" ]]; then
    env_args+=(--env-file "$ROOT/install.compose.env")
  fi
  docker compose "${file_args[@]}" "${profile_args[@]}" "${env_args[@]}" "$@"
}

CMD="${1:-status}"

INFRA=(db db-payment redis-0 redis-1 redis-2 redis-3 clickhouse)
SINGLE_VPS=(db redis-0 redis-1 redis-2 redis-3 clickhouse broker processor tracker-0 control)
INGEST_ONLY=(db redis-0 redis-1 redis-2 redis-3 broker processor tracker-0 control)
NETWORK_OPERATOR=(db db-payment redis-0 redis-1 redis-2 redis-3 clickhouse broker processor tracker-0 control)
SENTINEL=(redis-0 redis-0-replica sentinel-0 sentinel-1 sentinel-2)

case "$CMD" in
  infra | up-infra)
    espx_compose --profile infra up -d db db-payment redis-0 redis-1 redis-2 redis-3 redis-4 redis-5 clickhouse
    ;;
  full | up-full)
    echo "stack.sh: full runs single-vps monolith" >&2
    espx_compose --profile single_vps up -d "${SINGLE_VPS[@]}"
    espx_stack_hardening
    ;;
  legacy-full | up-legacy-full)
    echo "stack.sh: legacy-full removed; use single-vps or network-operator" >&2
    exit 1
    ;;
  single-vps | up-single-vps)
    local -a prof=(--profile single_vps)
    if espx_ingress_enabled; then
      prof+=(--profile ingress)
      bash "$SCRIPTS/install/render_ingress.sh"
    fi
    if espx_use_release_images; then
      espx_compose "${prof[@]}" pull tracker-0 processor control
      espx_compose "${prof[@]}" up -d --no-build "${SINGLE_VPS[@]}"
    else
      espx_compose "${prof[@]}" up -d "${SINGLE_VPS[@]}"
    fi
    espx_stack_hardening
    ;;
  ingest-only | up-ingest-only)
    CH_ENABLED=0 CONTROL_ENABLE_PAYMENT=0 CONTROL_ENABLE_BILLING=0 CONTROL_ENABLE_NOTIFIER=0 \
      CONTROL_ENABLE_MARGIN_GUARD=0 CONTROL_ENABLE_COST_SYNC=0 \
      espx_compose --profile ingest_only up -d "${INGEST_ONLY[@]}"
    espx_stack_hardening
    ;;
  network-operator | up-network-operator)
    espx_compose --profile network_operator up -d "${NETWORK_OPERATOR[@]}"
    espx_stack_hardening
    ;;
  analytics-ml | up-analytics-ml)
    espx_compose --profile analytics_ml --profile fraud-scorer up -d ivt-detector fraud-scorer clickhouse
    ;;
  sentinel | up-sentinel)
    espx_compose up -d "${SENTINEL[@]}"
    ;;
  multi-region | up-multi-region)
    sub="${2:-up}"
    case "$sub" in
      up)
        espx_compose --profile multi-region up -d region-proxy processor-1
        ;;
      broker)
        espx_compose up -d broker
        ;;
      down)
        espx_compose --profile multi-region stop region-proxy 2> /dev/null || true
        espx_compose stop broker 2> /dev/null || true
        ;;
      status)
        espx_compose --profile multi-region ps
        ;;
      *)
        echo "usage: $0 multi-region {up|broker|down|status}" >&2
        exit 2
        ;;
    esac
    ;;
  crypto | up-crypto)
    sub="${2:-up}"
    case "$sub" in
      up)
        espx_compose --profile crypto up -d db-payment control
        ;;
      down)
        espx_compose --profile crypto stop control 2> /dev/null || true
        ;;
      status)
        espx_compose --profile crypto ps
        ;;
      *)
        echo "usage: $0 crypto {up|down|status}" >&2
        exit 2
        ;;
    esac
    ;;
  down)
    espx_compose down
    ;;
  status)
    espx_compose ps
    echo "--- multi-region profile ---"
    espx_compose --profile multi-region ps
    ;;
  build)
    if espx_use_release_images; then
      echo "stack.sh: pulling release images (AD_EVENT_PROCESSOR_USE_RELEASE_IMAGES / AD_EVENT_PROCESSOR_APP_IMAGE)" >&2
      espx_compose pull tracker-0 processor control
    else
      espx_compose build
    fi
    bash "$SCRIPTS/dev/bpf_setup.sh" || echo "dev_stack: WARN bpf_setup failed (optional dev tooling)" >&2
    ;;
  bpf)
    bash "$SCRIPTS/dev/bpf_setup.sh"
    ;;
  probe)
    sub="${2:-status}"
    case "$sub" in
      start | stop | status | report)
        sudo bash "$SCRIPTS/dev/bpf_session.sh" "$sub" "${3:-}"
        ;;
      *)
        echo "usage: $0 probe {start|stop|status|report} [session_dir]" >&2
        exit 2
        ;;
    esac
    ;;
  *)
    echo "usage: $0 {infra|full|legacy-full|single-vps|ingest-only|network-operator|analytics-ml|sentinel|multi-region|crypto|down|status|build|bpf|probe}" >&2
    exit 2
    ;;
esac
