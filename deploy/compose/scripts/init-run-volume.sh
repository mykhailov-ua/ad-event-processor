#!/bin/sh
# Init run-volume directories for compose UDS and log spool paths.
# Runs once as run-dir-init service before db/redis/tracker (deploy/compose/docker-compose.yaml).
# Cross-ref: deploy/DEPLOY.md.
#
# Execution context:
# - busybox container; mounts ad_event_processor_run, logs, ch_spool volumes.
# - chmod -R 777 so host-network containers can create unix sockets as non-root.
#
# Env deps:
# - RUN_ROOT (default /run/ad-event-processor)
# - LOG_ROOT (default /var/log/ad-event-processor)
# - SPOOL_ROOT (default /var/spool/ad-event-processor/ch)
#
# Exit codes:
# - 0 on success; non-zero if mkdir or chmod fails (set -eu).
#
# Verify:
# go test ./deploy/compose/ -run TestInitRunVolumeScriptCreatesRedisDir -count=1
set -eu

RUN_ROOT="${RUN_ROOT:-/run/ad-event-processor}"
LOG_ROOT="${LOG_ROOT:-/var/log/ad-event-processor}"
SPOOL_ROOT="${SPOOL_ROOT:-/var/spool/ad-event-processor/ch}"

mkdir -p \
  "$RUN_ROOT/redis" \
  "$RUN_ROOT/postgresql" \
  "$RUN_ROOT/tracker" \
  "$RUN_ROOT/broker" \
  "$RUN_ROOT/region-proxy" \
  "$RUN_ROOT/clickhouse" \
  "$RUN_ROOT/control" \
  "$LOG_ROOT/offsets" \
  "$LOG_ROOT/offsets/fraud" \
  "$SPOOL_ROOT"

chmod -R 777 "$RUN_ROOT" "$LOG_ROOT" "$SPOOL_ROOT"
