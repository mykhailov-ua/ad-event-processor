#!/usr/bin/env bash
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
  echo "k3s_on: run with sudo: sudo bash scripts/dev/stack/k3s_on.sh" >&2
  exit 1
fi

systemctl unmask k3s
systemctl enable k3s
echo "k3s_on: unmasked and enabled; start with: sudo systemctl start k3s"
