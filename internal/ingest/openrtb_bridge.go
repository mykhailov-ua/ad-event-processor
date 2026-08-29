package ingest

import "ad-event-processor/internal/openrtb"

func parseDecimalMicro(b []byte) int64 {
	return openrtb.ParseDecimalMicro(b)
}

func ortbSlice(payload []byte, off int, ln uint8) []byte {
	return openrtb.OrtbSlice(payload, off, ln)
}

var ParseDealIDBytes = openrtb.ParseDealIDBytes
