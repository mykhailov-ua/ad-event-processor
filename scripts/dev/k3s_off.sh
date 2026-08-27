#!/usr/bin/env bash
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
  echo "k3s_off: run with sudo: sudo bash scripts/dev/k3s_off.sh" >&2
  exit 1
fi

if systemctl is-active --quiet k3s 2> /dev/null; then
  systemctl stop k3s
fi
if command -v k3s-killall.sh > /dev/null 2>&1; then
  k3s-killall.sh || true
fi
systemctl disable k3s
systemctl mask k3s

echo "k3s_off: stopped, disabled, and masked (k3s.service)"
systemctl is-active k3s 2> /dev/null || echo "k3s: not active"
