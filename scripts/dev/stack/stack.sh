#!/usr/bin/env bash
# Role: Dev compose orchestrator for ingest-only, single-vps, minimal, analytics-ml, and auxiliary profiles.
# Execution context: Operator laptop or shared dev host; sources scripts/lib and docker compose from repo root.
# Env knobs: REDIS_SHARD_COUNT (shards, max 6); INGEST_REDIS_SHARD_COUNT (ingest-only subset);
#   CH_ENABLED (0 default, 1 for clickhouse/minimal); COMPOSE_MEMORY_PROFILE (dev applies memory-dev overlay);
#   CPU_ISOLATION_ENABLED (1 adds cpu-isolation profile); EDGE_SYSCTL_AUTO_APPLY (1 applies host sysctl);
#   INGRESS_ENABLED (1 adds ingress profile); AD_EVENT_PROCESSOR_COMPOSE_EXTRA_FILES (comma compose overlays).
# Verify: bash scripts/dev/stack/stack.sh ingest-only && bash scripts/dev/stack/preflight.sh
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
source "$SCRIPTS/lib/installer_env.sh"
source "$SCRIPTS/lib/dev_bind_mounts.sh"
source "$SCRIPTS/lib/redis_topology.sh"
cd "$ROOT"

aed_read_env() {
  installer_read_env "$1"
}

aed_use_release_images() {
  installer_use_release_images
}

aed_ingress_enabled() {
  [[ "$(aed_read_env INGRESS_ENABLED)" == "1" ]]
}

aed_cpu_isolation_enabled() {
  [[ "$(aed_read_env CPU_ISOLATION_ENABLED)" == "1" ]]
}

aed_sysctl_auto_apply_enabled() {
  [[ "$(aed_read_env EDGE_SYSCTL_AUTO_APPLY)" == "1" ]]
}

aed_stack_hardening() {
  if [[ ! -x "$SCRIPTS/ops/sysctl.sh" ]]; then
    return 0
  fi
  if aed_sysctl_auto_apply_enabled; then
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

  if aed_cpu_isolation_enabled && command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then
    if ! bash "$SCRIPTS/ops/cpu_isolation.sh" verify 2> /dev/null; then
      echo "stack.sh: WARN cpu isolation verify failed (tracker-0 running with profile cpu-isolation?)" >&2
    fi
  fi
}

aed_append_compose_extra_file() {
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

aed_compose() {
  dev_prepare_compose_mounts
  # memory-dev overlay caps cgroup RAM on laptops; unset COMPOSE_MEMORY_PROFILE for compose defaults.
  if [[ "${COMPOSE_MEMORY_PROFILE:-dev}" == "dev" ]]; then
    aed_append_compose_extra_file "deploy/compose/docker-compose.memory-dev.yaml"
  else
    export COMPOSE_FILE="${COMPOSE_FILE:-$ROOT/docker-compose.yaml}"
  fi
  local -a env_args=()
  local -a file_args=(-f "$ROOT/docker-compose.yaml")
  local -a profile_args=()
  if aed_cpu_isolation_enabled; then
    file_args+=(-f "$ROOT/deploy/compose/docker-compose.cpu-isolation.yaml")
    profile_args+=(--profile cpu-isolation)
  fi
  if aed_use_release_images; then
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

# REDIS_SHARD_COUNT drives redis-0..N compose services; INGEST_REDIS_SHARD_COUNT may start fewer on ingest-only.
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
MINIMAL=(db redis-0 broker processor tracker-0 control clickhouse)
INGEST_DEV_COMPOSE=deploy/compose/docker-compose.control-dev.yaml
MINIMAL_COMPOSE=deploy/compose/docker-compose.minimal.yaml
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

aed_stop_vps_extras() {
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
    aed_compose --profile infra up -d "${INFRA[@]}"
    ;;
  clickhouse | up-clickhouse)
    # ClickHouse is explicit opt-in; never started by ingest-only or full without this subcommand.
    echo "stack.sh: ClickHouse is heavy (RAM/IO). Use only for hotfix, P0 e2e, or IOPS drills." >&2
    CH_ENABLED=1 aed_compose --profile single_vps up -d clickhouse
    ;;
  full | up-full)
    echo "stack.sh: full runs single-vps monolith (no ClickHouse; use: stack.sh clickhouse)" >&2
    CH_ENABLED=0 aed_compose --profile single_vps up -d "${SINGLE_VPS[@]}"
    aed_stack_hardening
    ;;
  single-vps | up-single-vps)
    prof=(--profile single_vps)
    if aed_ingress_enabled; then
      prof+=(--profile ingress)
      bash "$SCRIPTS/install/render_ingress.sh"
    fi
    if aed_use_release_images; then
      aed_compose "${prof[@]}" pull tracker-0 processor control
      CH_ENABLED=0 aed_compose "${prof[@]}" up -d --no-build "${SINGLE_VPS[@]}"
    else
      CH_ENABLED=0 aed_compose "${prof[@]}" up -d "${SINGLE_VPS[@]}"
    fi
    aed_stack_hardening
    ;;
  ingest-only | up-ingest-only)
    # Canonical laptop path: no CH, cold-path workers off, control-dev overlay for payment stubs.
    aed_stop_vps_extras
    CH_ENABLED=0 CONTROL_ENABLE_PAYMENT=0 CONTROL_ENABLE_BILLING=0 CONTROL_ENABLE_NOTIFIER=0 \
      CONTROL_ENABLE_MARGIN_GUARD=0 CONTROL_ENABLE_COST_SYNC=0 \
      AD_EVENT_PROCESSOR_COMPOSE_EXTRA_FILES="$INGEST_DEV_COMPOSE" \
      aed_compose --profile ingest_only up -d "${INGEST_ONLY[@]}"
    aed_stop_vps_extras
    aed_stack_hardening
    ;;
  minimal | up-minimal)
    echo "stack.sh: minimal profile runs tracker+control+PG+single Redis+CH; antifraud ML disabled." >&2
    aed_stop_vps_extras
    if [[ -f "$ROOT/deploy/compose/minimal.stack.env.example" ]]; then
      echo "stack.sh: merge deploy/compose/minimal.stack.env.example into .env for stable defaults." >&2
    fi
    CH_ENABLED=1 REDIS_SHARD_COUNT=1 INGEST_REDIS_SHARD_COUNT=1 \
      FRAUD_SCORING_ENABLED=false FRAUD_MICROBATCH_ENABLED=false \
      CONTROL_ENABLE_PAYMENT=0 CONTROL_ENABLE_BILLING=0 CONTROL_ENABLE_NOTIFIER=0 \
      CONTROL_ENABLE_MARGIN_GUARD=0 CONTROL_ENABLE_COST_SYNC=0 \
      CONTROL_ENABLE_PLATFORM_CAMPAIGN_SYNC=0 \
      AD_EVENT_PROCESSOR_COMPOSE_EXTRA_FILES="$MINIMAL_COMPOSE" \
      aed_compose --profile minimal up -d "${MINIMAL[@]}"
    aed_stop_vps_extras
    aed_stack_hardening
    ;;
  network-operator | up-network-operator)
    CH_ENABLED=0 aed_compose --profile network_operator up -d "${NETWORK_OPERATOR[@]}"
    aed_stack_hardening
    ;;
  analytics-ml | up-analytics-ml)
    # analytics_ml profile requires ClickHouse for fraud-scorer and ivt-detector sidecars.
    aed_compose --profile analytics_ml --profile fraud-scorer up -d ivt-detector fraud-scorer clickhouse
    ;;
  sentinel | up-sentinel)
    aed_compose up -d "${SENTINEL[@]}"
    ;;
  multi-region | up-multi-region)
    sub="${2:-up}"
    case "$sub" in
      up)
        aed_compose --profile multi-region up -d region-proxy processor-1
        ;;
      broker)
        aed_compose up -d broker
        ;;
      down)
        aed_compose --profile multi-region stop region-proxy 2> /dev/null || true
        aed_compose stop broker 2> /dev/null || true
        ;;
      status)
        aed_compose --profile multi-region ps
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
        aed_compose --profile crypto up -d db-payment control
        ;;
      down)
        aed_compose --profile crypto stop control 2> /dev/null || true
        ;;
      status)
        aed_compose --profile crypto ps
        ;;
      *)
        echo "usage: $0 crypto {up|down|status}" >&2
        exit 2
        ;;
    esac
    ;;
  down)
    aed_compose down
    ;;
  status)
    aed_compose ps
    echo "multi-region profile"
    aed_compose --profile multi-region ps
    ;;
  build)
    if aed_use_release_images; then
      echo "stack.sh: pulling release images (AD_EVENT_PROCESSOR_USE_RELEASE_IMAGES / AD_EVENT_PROCESSOR_APP_IMAGE)" >&2
      aed_compose pull tracker-0 processor control
    else
      aed_compose build
    fi
    bash "$SCRIPTS/dev/stack/bpf_setup.sh" || echo "dev_stack: WARN bpf_setup failed (optional dev tooling)" >&2
    ;;
  bpf)
    bash "$SCRIPTS/dev/stack/bpf_setup.sh"
    ;;
  probe)
    sub="${2:-status}"
    case "$sub" in
      start | stop | status | report)
        sudo bash "$SCRIPTS/dev/stack/bpf_session.sh" "$sub" "${3:-}"
        ;;
      *)
        echo "usage: $0 probe {start|stop|status|report} [session_dir]" >&2
        exit 2
        ;;
    esac
    ;;
  *)
    echo "usage: $0 {infra|clickhouse|full|single-vps|ingest-only|minimal|network-operator|analytics-ml|sentinel|multi-region|crypto|down|status|build|bpf|probe}" >&2
    exit 2
    ;;
esac
