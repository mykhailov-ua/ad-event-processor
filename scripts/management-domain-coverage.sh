#!/usr/bin/env bash
# Per-domain business-logic coverage gate for internal/management (GAP-ENG-01).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
COVER="${TMPDIR:-/tmp}/espx-mgmt-domain.cover"

cd "$ROOT"

go test ./internal/management -short -count=1 \
	-run '^(TestDomainRegistry|TestBoundaryDTO|TestBilling_|TestOperation_|TestRecon_|TestNode_|TestCampaign_|TestCore_|TestScoreNode|TestForecast_|TestPlatform_|TestMapServiceError|TestParseMoneyMicro|TestLeaseFencing|TestAuthHandler|TestCORSMiddleware|TestCSRFMiddleware|TestSettlementGRPC)' \
	-coverprofile="$COVER"

ESPX_MGMT_COVER_PROFILE="$COVER" go test ./internal/management -run TestDomainBusinessLogicCoverage -count=1

echo "management domain coverage: PASS"
