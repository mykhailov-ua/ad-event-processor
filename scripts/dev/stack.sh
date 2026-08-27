#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/installer_env.sh"
source "$SCRIPTS/lib/dev_bind_mounts.sh"
source "$SCRIPTS/lib/redis_topology.sh"
cd "$ROOT"

ad_event_processor_read_env() {
  installer_read_env "$1"
}

ad_event_processor_use_release_images() {
  installer_use_release_images
}

ad_event_processor_ingress_enabled() {
  [[ "$(ad_event_processor_read_env INGRESS_ENABLED)" == "1" ]]
}

ad_event_processor_cpu_isolation_enabled() {
  [[ "$(ad_event_processor_read_env CPU_ISOLATION_ENABLED)" == "1" ]]
}

ad_event_processor_sysctl_auto_apply_enabled() {
  [[ "$(ad_event_processor_read_env EDGE_SYSCTL_AUTO_APPLY)" == "1" ]]
}

ad_event_processor_stack_hardening() {
  if [[ ! -x "$SCRIPTS/ops/sysctl.sh" ]]; then
    return 0
  fi
  if ad_event_processor_sysctl_auto_apply_enabled; then
    if [[ "$(id -u)" -eq 0 ]]; then
      echo "stack.sh: applying host sysctl (EDGE_SYSCTL_AUTO_APPLY=1)" >&2
      bash "$SCRIPTS/ops/sysctl.sh" apply || echo "stack.sh: WARN sysctl apply failed" >&2
    else
      if ! bash "$SCRIPTS/ops/sysctl.sh" verify 2> /dev/null; then
        echo "stack.sh: WARN sysctl not applied - run: sudo bash scripts/ops/sysctl.sh apply" >&2
        echo "stack.sh: WARN recreate listeners after somaxconn change (see deploy/edge/99-ad-event-processor-sysctl.conf)" >&2
      fi
    fi
  else
    bash "$SCRIPTS/ops/sysctl.sh" verify 2> /dev/null \
      || echo "stack.sh: hint: EDGE_SYSCTL_AUTO_APPLY=1 or sudo bash scripts/ops/sysctl.sh apply" >&2
  fi

  if ad_event_processor_cpu_isolation_enabled && command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then
    if ! bash "$SCRIPTS/ops/cpu_isolation.sh" verify 2> /dev/null; then
      echo "stack.sh: WARN cpu isolation verify failed (tracker-0 running with profile cpu-isolation?)" >&2
    fi
  fi
}

ad_event_processor_append_compose_extra_file() {
  local file="$1"
  if [[ -z "${AD_EVENT_PROCESSOR_COMPOSE_EXTRA_FILES:-}" ]]; then
    export AD_EVENT_PROCESSOR_COMPOSE_EXTRA_FILES="$file"
    return 0
  fi
  if [[ ",${AD_EVENT_PROCESSOR_COMPOSE_EXTRA_FILES}," == *",$file,"* ]]; then
    return 0
  fi
  export AD_EVENT_PROCESSOR_COMPOSE_EXTRA_FILES="${AD_EVENT_PROCESSOR_COMPOSE_EXTRA_FILES},${file}"
}

ad_event_processor_compose() {
  dev_prepare_compose_mounts
  if [[ "${COMPOSE_MEMORY_PROFILE:-dev}" == "dev" ]]; then
    ad_event_processor_append_compose_extra_file "deploy/compose/docker-compose.memory-dev.yaml"
  fi
  local -a env_args=()
  local -a file_args=(-f "$ROOT/docker-compose.yaml")
  local -a profile_args=()
  if ad_event_processor_cpu_isolation_enabled; then
    file_args+=(-f "$ROOT/deploy/compose/docker-compose.cpu-isolation.yaml")
    profile_args+=(--profile cpu-isolation)
  fi
  if ad_event_processor_use_release_images; then
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
  if [[ -n "${AD_EVENT_PROCESSOR_COMPOSE_EXTRA_FILES:-}" ]]; then
    local -a extra_files=()
    IFS=',' read -r -a extra_files <<< "$AD_EVENT_PROCESSOR_COMPOSE_EXTRA_FILES"
    for extra in "${extra_files[@]}"; do
      file_args+=(-f "$ROOT/$extra")
    done
  fi
  docker compose --project-directory "$ROOT" "${file_args[@]}" "${profile_args[@]}" "${env_args[@]}" "$@"
  local rc=$?
  dev_finalize_compose_mounts
  return "$rc"
}

CMD="${1:-status}"

REDIS_SHARD_COUNT="$(redis_topology_count)"
read -ra REDIS_SHARDS <<< "$(redis_topology_services "$REDIS_SHARD_COUNT")"

INGEST_REDIS_SHARD_COUNT="${INGEST_REDIS_SHARD_COUNT:-2}"
if [[ "$INGEST_REDIS_SHARD_COUNT" -gt "$REDIS_SHARD_COUNT" ]]; then
  INGEST_REDIS_SHARD_COUNT="$REDIS_SHARD_COUNT"
fi
read -ra INGEST_REDIS_SHARDS <<< "$(redis_topology_services "$INGEST_REDIS_SHARD_COUNT")"

INFRA=(db db-payment "${REDIS_SHARDS[@]}")
SINGLE_VPS=(db "${REDIS_SHARDS[@]}" broker processor tracker-0 control)
INGEST_ONLY=(db "${INGEST_REDIS_SHARDS[@]}" broker processor tracker-0 control)
INGEST_DEV_COMPOSE=deploy/compose/docker-compose.control-dev.yaml
VPS_EXTRA_SERVICES=(
  db-payment clickhouse nginx tracker-1 tracker-2 tracker-3
  prometheus grafana loki promtail
)
max_shards="$(redis_topology_max_shards)"
i="$INGEST_REDIS_SHARD_COUNT"
while [[ "$i" -lt "$max_shards" ]]; do
  VPS_EXTRA_SERVICES+=("redis-$i")
  i=$((i + 1))
done

ad_event_processor_stop_vps_extras() {
  if ! command -v docker > /dev/null 2>&1; then
    return 0
  fi
  local -a file_args=(-f "$ROOT/docker-compose.yaml")
  local -a env_args=()
  if [[ -f "$ROOT/.env" ]]; then
    env_args+=(--env-file "$ROOT/.env")
  fi
  if [[ -f "$ROOT/install.compose.env" ]]; then
    env_args+=(--env-file "$ROOT/install.compose.env")
  fi
  docker compose --project-directory "$ROOT" "${file_args[@]}" "${env_args[@]}" stop "${VPS_EXTRA_SERVICES[@]}" 2> /dev/null \
    || true
}

NETWORK_OPERATOR=(db db-payment redis-0 redis-1 redis-2 redis-3 broker processor tracker-0 control)
SENTINEL=(redis-0 redis-0-replica sentinel-0 sentinel-1 sentinel-2)

case "$CMD" in
  infra | up-infra)
    ad_event_processor_compose --profile infra up -d "${INFRA[@]}"
    ;;
  clickhouse | up-clickhouse)
    echo "stack.sh: ClickHouse is heavy (RAM/IO). Use only for hotfix, P0 e2e, or IOPS drills." >&2
    CH_ENABLED=1 ad_event_processor_compose --profile single_vps up -d clickhouse
    ;;
  full | up-full)
    echo "stack.sh: full runs single-vps monolith (no ClickHouse; use: stack.sh clickhouse)" >&2
    CH_ENABLED=0 ad_event_processor_compose --profile single_vps up -d "${SINGLE_VPS[@]}"
    ad_event_processor_stack_hardening
    ;;
  single-vps | up-single-vps)
    prof=(--profile single_vps)
    if ad_event_processor_ingress_enabled; then
      prof+=(--profile ingress)
      bash "$SCRIPTS/install/render_ingress.sh"
    fi
    if ad_event_processor_use_release_images; then
      ad_event_processor_compose "${prof[@]}" pull tracker-0 processor control
      CH_ENABLED=0 ad_event_processor_compose "${prof[@]}" up -d --no-build "${SINGLE_VPS[@]}"
    else
      CH_ENABLED=0 ad_event_processor_compose "${prof[@]}" up -d "${SINGLE_VPS[@]}"
    fi
    ad_event_processor_stack_hardening
    ;;
  ingest-only | up-ingest-only)
    ad_event_processor_stop_vps_extras
    CH_ENABLED=0 CONTROL_ENABLE_PAYMENT=0 CONTROL_ENABLE_BILLING=0 CONTROL_ENABLE_NOTIFIER=0 \
      CONTROL_ENABLE_MARGIN_GUARD=0 CONTROL_ENABLE_COST_SYNC=0 \
      AD_EVENT_PROCESSOR_COMPOSE_EXTRA_FILES="$INGEST_DEV_COMPOSE" \
      ad_event_processor_compose --profile ingest_only up -d "${INGEST_ONLY[@]}"
    ad_event_processor_stop_vps_extras
    ad_event_processor_stack_hardening
    ;;
  network-operator | up-network-operator)
    CH_ENABLED=0 ad_event_processor_compose --profile network_operator up -d "${NETWORK_OPERATOR[@]}"
    ad_event_processor_stack_hardening
    ;;
  analytics-ml | up-analytics-ml)
    ad_event_processor_compose --profile analytics_ml --profile fraud-scorer up -d ivt-detector fraud-scorer clickhouse
    ;;
  sentinel | up-sentinel)
    ad_event_processor_compose up -d "${SENTINEL[@]}"
    ;;
  multi-region | up-multi-region)
    sub="${2:-up}"
    case "$sub" in
      up)
        ad_event_processor_compose --profile multi-region up -d region-proxy processor-1
        ;;
      broker)
        ad_event_processor_compose up -d broker
        ;;
      down)
        ad_event_processor_compose --profile multi-region stop region-proxy 2> /dev/null || true
        ad_event_processor_compose stop broker 2> /dev/null || true
        ;;
      status)
        ad_event_processor_compose --profile multi-region ps
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
        ad_event_processor_compose --profile crypto up -d db-payment control
        ;;
      down)
        ad_event_processor_compose --profile crypto stop control 2> /dev/null || true
        ;;
      status)
        ad_event_processor_compose --profile crypto ps
        ;;
      *)
        echo "usage: $0 crypto {up|down|status}" >&2
        exit 2
        ;;
    esac
    ;;
  down)
    ad_event_processor_compose down
    ;;
  status)
    ad_event_processor_compose ps
    echo "multi-region profile"
    ad_event_processor_compose --profile multi-region ps
    ;;
  build)
    if ad_event_processor_use_release_images; then
      echo "stack.sh: pulling release images (AD_EVENT_PROCESSOR_USE_RELEASE_IMAGES / AD_EVENT_PROCESSOR_APP_IMAGE)" >&2
      ad_event_processor_compose pull tracker-0 processor control
    else
      ad_event_processor_compose build
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
    echo "usage: $0 {infra|clickhouse|full|single-vps|ingest-only|network-operator|analytics-ml|sentinel|multi-region|crypto|down|status|build|bpf|probe}" >&2
    exit 2
    ;;
esac
