#!/bin/sh
set -eu

CONF=/tmp/sentinel.conf
: "${REDIS_PASSWORD:?REDIS_PASSWORD is required}"
: "${REDIS_SHARD_COUNT:=4}"

cat > "$CONF" << EOF
port 26379
dir /tmp
protected-mode no
sentinel resolve-hostnames yes
sentinel announce-hostnames yes
EOF

i=0
while [ "$i" -lt "$REDIS_SHARD_COUNT" ]; do
  cat >> "$CONF" << EOF
sentinel monitor ad-event-processor-shard-${i} redis-${i} 6379 2
sentinel auth-pass ad-event-processor-shard-${i} ${REDIS_PASSWORD}
sentinel down-after-milliseconds ad-event-processor-shard-${i} 5000
sentinel failover-timeout ad-event-processor-shard-${i} 10000
sentinel parallel-syncs ad-event-processor-shard-${i} 1
EOF
  i=$((i + 1))
done

exec redis-sentinel "$CONF" --sentinel
