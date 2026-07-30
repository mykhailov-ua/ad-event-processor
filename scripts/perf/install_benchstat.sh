#!/usr/bin/env bash
set -euo pipefail

if command -v benchstat >/dev/null 2>&1; then
	exit 0
fi

go install golang.org/x/perf/cmd/benchstat@latest
