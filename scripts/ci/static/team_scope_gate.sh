#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"

fail=0
check() {
  if ! rg -q "$1" "$2"; then
    echo "team_scope_gate: missing $3 in $2" >&2
    fail=1
  fi
}

check 'campaign\.ApplyListScopeFilter' "$ROOT/internal/campaign/runtime/ops.go" 'ApplyListScopeFilter in listCampaigns'
check 'teamscope\.ScopedCampaignIDsQuery' "$ROOT/internal/campaign/scoped_campaign_ids.go" 'ScopedCampaignIDsQuery'
check 'mask\.TouchesProtectedFields' "$ROOT/internal/campaign/runtime/ops.go" 'masked mutation deny'
check 'GET /api/v1/team/teams' "$ROOT/internal/platformadmin/teams_handlers.go" 'team teams route'

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

echo "team_scope_gate: ok"
