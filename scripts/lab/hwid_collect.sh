#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

export HWID_LAB_COLLECT=1
go test ./internal/licensing/ -run '^TestHWID_LabCollectPrint$' -count=1 -v
