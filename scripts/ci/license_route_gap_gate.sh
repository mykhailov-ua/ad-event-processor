#!/usr/bin/env bash

set -euo pipefail
cd "$(dirname "$0")/../.."
go test ./internal/controlplane/ -run 'TestLicense_' -count=1
echo "license_route_gap_gate: OK"
