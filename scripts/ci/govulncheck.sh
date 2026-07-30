#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if ! command -v govulncheck >/dev/null 2>&1; then
	echo "Installing govulncheck..."
	go install golang.org/x/vuln/cmd/govulncheck@latest
fi

GOPATH="$(go env GOPATH)"
if [ -z "$GOPATH" ]; then
	GOPATH="$HOME/go"
fi

"$GOPATH/bin/govulncheck" ./...
