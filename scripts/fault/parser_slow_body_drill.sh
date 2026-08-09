#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

CONNECTIONS=64
RATE=1
DURATION=120s

while [[ $# -gt 0 ]]; do
	case "$1" in
	--connections=*)
		CONNECTIONS="${1#*=}"
		shift
		;;
	--rate=*)
		RATE="${1#*=}"
		shift
		;;
	--duration=*)
		DURATION="${1#*=}"
		shift
		;;
	*)
		echo "usage: $0 [--connections=N] [--rate=bytes_per_sec] [--duration=120s]" >&2
		exit 2
		;;
	esac
done

echo "parser-slow-body: unit proof (PS-G01 spin close)"
go test ./internal/ingestion/ -run='TestChaos_ParserSecurity_PS_G01|TestHTTP1Incomplete' -count=1 -timeout=5m -v

echo "parser-slow-body: integration stub connections=${CONNECTIONS} rate=${RATE}B/s duration=${DURATION}"
echo "fault_proof fault=parser_slow_body_drill gap=stub connections=${CONNECTIONS} rate=${RATE} duration=${DURATION} note=run_against_live_tracker_for_p99_gate"
