#!/usr/bin/env bash
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"
go test ./internal/loadreport/ -run TestWriteBPFReport_diskGateFixture -count=1
