#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"
cd "$ROOT"

BENCH_PATTERN='Benchmark(AdsPacketHandlerProto$$|AdsPacketHandlerProto_NoExtra|AdsPacketHandlerProto_ExtraBytes|HotPath_|TrackRequest_ParseJSON|CompositeRouting_Protobuf|Auction$$|HTTP1Parse$$|HTTP1DFA_|HTTP2DFA_|HTTP2DecodeFrame$$|HTTP3DFA_|HTTP3VarintDecode$$|ParseOpenRTB26|RunOpenRTBExchangeParsed|OpenRTB26_exchangeGnet|WriteOpenRTB26|ParseTgQuery|TgClickRedirectGnet_E2E|BuildTgRedirectLocation|ParseTgBidRequest|TrackerToBroker$$|ParseClickQuery|CIDR_LPM_Lookup_IPv4|CIDR_LPM_Lookup_IPv6|CIDR_MatchBranch_SafeView|ClickProxy_Stream$$|ClickProxy_BuildUpstreamURL$$)'

BENCH_COUNT=10
if [[ "${PERF_GATE_STRICT:-}" != "true" ]]; then
	BENCH_COUNT=2
fi

export GOMAXPROCS=1
exec go test -run='^$' \
	-bench="$BENCH_PATTERN" \
	-benchmem \
	-benchtime=200ms \
	-count="$BENCH_COUNT" \
	-cpu=1 \
	./internal/ingestion ./internal/rtb
