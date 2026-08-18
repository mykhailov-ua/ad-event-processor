#!/bin/sh
set -eu

if [ -z "${INGRESS_INTERFACE:-}" ]; then
  echo "edge-xdp: INGRESS_INTERFACE is required" >&2
  exit 1
fi

BPF_PIN_DIR="${BPF_PIN_DIR:-/sys/fs/bpf/ad-event-processor}"
BLOCKLIST_MAP="${BPF_BLOCKLIST_MAP:-${BPF_PIN_DIR}/blocklist_v4}"

export BPF_PIN_DIR

/usr/local/bin/edge-xdp &
xdp_pid=$!

i=0
while [ "$i" -lt 50 ]; do
  if [ -e "$BLOCKLIST_MAP" ]; then
    break
  fi
  i=$((i + 1))
  sleep 0.2
done

if [ ! -e "$BLOCKLIST_MAP" ]; then
  echo "edge-xdp: timed out waiting for pinned blocklist map at $BLOCKLIST_MAP" >&2
  kill "$xdp_pid" 2> /dev/null || true
  wait "$xdp_pid" 2> /dev/null || true
  exit 1
fi

cleanup() {
  kill "$xdp_pid" 2> /dev/null || true
  wait "$xdp_pid" 2> /dev/null || true
}
trap cleanup EXIT INT TERM

exec /usr/local/bin/edge-bpf-sync
