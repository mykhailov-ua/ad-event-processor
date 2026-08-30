#!/bin/sh
# Role: Assert REDIS_ADDRS entry count matches REDIS_SHARD_COUNT in an env file.
# Execution context: Operator pre-deploy; reads env file path argument (default .env).
# Env knobs: REDIS_SHARD_COUNT (from env or file, default 2); REDIS_ADDRS (comma-separated).
# Verify: bash scripts/ops/verify_redis_topology.sh .env
set -eu

ENV_FILE="${1:-.env}"

# REDIS_ADDRS count must equal REDIS_SHARD_COUNT (production requires 4; dev default 2).
EXPECTED="${REDIS_SHARD_COUNT:-}"
if [ -z "$EXPECTED" ]; then
  EXPECTED="$(grep -E '^REDIS_SHARD_COUNT=' "$ENV_FILE" 2> /dev/null | tail -1 | cut -d= -f2- | tr -d '"' | tr -d "'" | tr -d ' ')"
fi
if [ -z "$EXPECTED" ]; then
  EXPECTED=2
fi

if [ ! -f "$ENV_FILE" ]; then
  echo "verify_redis_topology: missing env file: $ENV_FILE" >&2
  exit 1
fi

REDIS_ADDRS="$(grep -E '^REDIS_ADDRS=' "$ENV_FILE" | tail -1 | cut -d= -f2- | tr -d '"' | tr -d "'")"
if [ -z "$REDIS_ADDRS" ]; then
  echo "verify_redis_topology: REDIS_ADDRS not set in $ENV_FILE" >&2
  exit 1
fi

COUNT=0
OLDIFS="$IFS"
IFS=,
for addr in $REDIS_ADDRS; do
  addr="$(echo "$addr" | tr -d ' ')"
  [ -n "$addr" ] || continue
  COUNT=$((COUNT + 1))
done
IFS="$OLDIFS"

if [ "$COUNT" -ne "$EXPECTED" ]; then
  echo "verify_redis_topology: expected $EXPECTED Redis shards, got $COUNT (REDIS_ADDRS=$REDIS_ADDRS)" >&2
  exit 1
fi

echo "verify_redis_topology: OK ($COUNT shards)"
