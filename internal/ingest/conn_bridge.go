package ingest

import (
	"ad-event-processor/internal/ingest/conn"
	"ad-event-processor/internal/track"
)

func parseTLSFingerprintFeed(data []byte) (ja3, ja4 []uint32) {
	return conn.ParseTLSFingerprintFeed(data)
}

func parseTLSFingerprintAllowFeed(data []byte) (ja3, ja4 []uint32) {
	return conn.ParseTLSFingerprintAllowFeed(data)
}

func checkBezierBot(events []track.SafePageVerifyEvent) string {
	return track.CheckBezierBot(events)
}

func tlsHashBlocked(sorted []uint32, h uint32) bool {
	return conn.TLSHashBlocked(sorted, h)
}

const (
	residentialIntelFeedFileName = conn.ResidentialIntelFeedFileName
	residentialIntelRedisPrefix  = conn.ResidentialIntelRedisPrefix
)

type residentialIntelRedisEntry = conn.ResidentialIntelRedisEntry
