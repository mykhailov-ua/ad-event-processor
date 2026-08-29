#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

COVER="${TMPDIR:-/tmp}/ad-event-processor-mgmt-domain.cover"

go test ./internal/controlplane -short -count=1 \
  -run '^(TestDomainRegistry|TestBoundaryDTO|TestBilling_|TestOperation_|TestRecon_|TestNode_|TestCampaign_|TestCore_|TestScoreNode|TestForecast_|TestPlatform_|TestMapServiceError|TestParseMoneyMicro|TestLeaseFencing|TestAuthHandler|TestCORSMiddleware|TestCSRFMiddleware|TestSettlementGRPC)' \
  -coverprofile="$COVER"

AD_EVENT_PROCESSOR_MGMT_COVER_PROFILE="$COVER" go test ./internal/controlplane -run TestDomainBusinessLogicCoverage -count=1

echo "management domain coverage: PASS"
