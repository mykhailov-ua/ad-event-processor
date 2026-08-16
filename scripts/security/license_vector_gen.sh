#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

WRITE_HWID_VECTORS=1 go test ./internal/licensing/ -run TestGenHWIDVectorArtifacts -count=1
WRITE_MCK_VECTORS=1 go test ./internal/licensing/ -run TestGenMCKVectorArtifacts -count=1
