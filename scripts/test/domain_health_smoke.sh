#!/usr/bin/env bash
# GMA-M5 domain health smoke: Cloudflare mock tests + reputation + ban integration.
# Skip (no go): not applicable.
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if ! command -v go > /dev/null 2>&1; then
  echo "skip (go not found)"
  exit 0
fi

echo "domain_health_smoke: pkg/domainhealth"
go test ./pkg/domainhealth/ -count=1

echo "domain_health_smoke: Cloudflare + ban integration"
if ! docker info > /dev/null 2>&1; then
  echo "skip (Docker unavailable) integration tests"
  go test ./internal/controlplane/ -run 'TestCloudflare' -count=1
  exit 0
fi

go test ./internal/controlplane/ -run 'TestCloudflare|TestDomainHealth_|TestApplyReputation' -count=1
go test ./internal/ingestion/ -run 'TestDomainPoolTable_FallbackHost|TestClickRedirect_DomainRotation' -count=1

echo "domain_health_smoke: OK"
