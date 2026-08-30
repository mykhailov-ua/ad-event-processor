#!/usr/bin/env bash
# Role: Set CPU governor to performance for stable microbench and load-test runs.
# Execution context: Linux host; requires sudo for cpufreq and optional cpupower.
# Env knobs: none.
# Verify: bash scripts/perf/stabilize_cpu.sh
set -euo pipefail

if [[ -d /sys/devices/system/cpu/cpu0/cpufreq ]]; then
  echo "performance" | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor > /dev/null || true
fi
if command -v cpupower > /dev/null 2>&1; then
  sudo cpupower frequency-set -g performance > /dev/null || true
fi
