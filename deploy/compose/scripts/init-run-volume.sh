#!/bin/sh
set -eu

RUN_ROOT="${RUN_ROOT:-/run/ad-event-processor}"

mkdir -p \
	"$RUN_ROOT/redis" \
	"$RUN_ROOT/postgresql" \
	"$RUN_ROOT/tracker" \
	"$RUN_ROOT/broker" \
	"$RUN_ROOT/region-proxy" \
	"$RUN_ROOT/clickhouse" \
	"$RUN_ROOT/control"

chmod -R 777 "$RUN_ROOT"
