#!/usr/bin/env bash
# Role: Unmask and enable k3s systemd unit for optional local Kubernetes experiments.
# Execution context: Dev host as root; does not start k3s (operator runs systemctl start k3s).
# Env knobs: none.
# Verify: sudo bash scripts/dev/stack/k3s_on.sh && systemctl is-enabled k3s
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
  echo "k3s_on: run with sudo: sudo bash scripts/dev/stack/k3s_on.sh" >&2
  exit 1
fi

systemctl unmask k3s
systemctl enable k3s
echo "k3s_on: unmasked and enabled; start with: sudo systemctl start k3s"
