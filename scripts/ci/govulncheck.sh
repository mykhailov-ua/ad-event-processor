#!/usr/bin/env bash
set -euo pipefail

# Role: govulncheck vulnerability scan.
# Execution context: CI optional.
# Invariants/contracts enforced: Known vulns fail when strict.
# Verify: bash scripts/ci/govulncheck.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if ! command -v govulncheck > /dev/null 2>&1; then
  echo "Installing govulncheck..."
  go install golang.org/x/vuln/cmd/govulncheck@latest
fi

GOPATH="$(go env GOPATH)"
if [ -z "$GOPATH" ]; then
  GOPATH="$HOME/go"
fi

"$GOPATH/bin/govulncheck" ./...
