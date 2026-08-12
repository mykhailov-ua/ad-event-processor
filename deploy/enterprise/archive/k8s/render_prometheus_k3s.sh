#!/usr/bin/env bash
set -euo pipefail

ARCHIVE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$ARCHIVE/../../.." && pwd)"
cd "$ROOT"

export KUBECONFIG="${KUBECONFIG:-$HOME/.kube/config-espx}"
OUT="${1:-$ROOT/deploy/monitoring/prometheus-k3s.rendered.yaml}"

NODE_IP="$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')"
sed "s/__NODE_IP__/${NODE_IP}/g" "$ARCHIVE/prometheus-k3s.yaml" >"$OUT"
printf 'render-prometheus-k3s: wrote %s (node=%s)\n' "$OUT" "$NODE_IP"
