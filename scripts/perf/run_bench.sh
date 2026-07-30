#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

if [[ $
	echo "usage: $0 <bench_regex> <package...>" >&2
	exit 2
fi

PATTERN="$1"
shift

export GOMAXPROCS=1
exec go test -run='^$' \
	-bench="$PATTERN" \
	-benchmem \
	-benchtime=200ms \
	-count=10 \
	-cpu=1 \
	"$@"
