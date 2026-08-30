#!/usr/bin/env bash
set -euo pipefail

# Role: Static gate: Management domain test coverage floor.
# Execution context: CI merge-pr-fast via pr_fast unless noted.
# Invariants/contracts enforced: Non-zero exit on contract violation; no silent pass on failure.
# Verify: bash scripts/ci/static/management_domain_coverage.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib/paths.sh"
cd "$ROOT"

COVER="${TMPDIR:-/tmp}/ad-event-processor-mgmt-domain.cover"

go test ./internal/controlplane -short -count=1 \
  -run '^(TestDomainRegistry|TestBoundaryDTO|TestBilling_|TestOperation_|TestRecon_|TestNode_|TestCampaign_|TestCore_|TestScoreNode|TestForecast_|TestPlatform_|TestMapServiceError|TestParseMoneyMicro|TestLeaseFencing|TestAuthHandler|TestCORSMiddleware|TestCSRFMiddleware|TestSettlementGRPC)' \
  -coverprofile="$COVER"

AD_EVENT_PROCESSOR_MGMT_COVER_PROFILE="$COVER" go test ./internal/controlplane -run TestDomainBusinessLogicCoverage -count=1

echo "management domain coverage: PASS"
