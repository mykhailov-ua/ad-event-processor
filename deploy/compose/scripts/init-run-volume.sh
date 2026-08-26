#!/bin/sh
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
