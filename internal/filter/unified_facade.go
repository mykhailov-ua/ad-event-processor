package filter

import (
	"context"
	"time"

	"ad-event-processor/internal/domain"
)

func AcceptEncodingBrowserMismatch(ua string, encFlags, encSet uint8) bool {
	return acceptEncodingBrowserMismatch(ua, encFlags, encSet)
}

func SecFetchAnomaly(ua string, present, mode, dest uint8) bool {
	return secFetchAnomaly(ua, present, mode, dest)
}

func ClientHintsPlatformMismatch(ua, platform string, mobile uint8) bool {
	return clientHintsPlatformMismatch(ua, platform, mobile)
}

func TLSALPNBrowserMismatch(ua, alpn string) bool {
	return tlsALPNBrowserMismatch(ua, alpn)
}

func HTTP1HeaderOrderMismatch(ua string, order []uint8, count uint8, secFetchPresent uint8) bool {
	return http1HeaderOrderMismatch(ua, order, count, secFetchPresent)
}

func H2SettingsAnomaly(ua string, flags uint8, enablePush uint8, initialWindow, windowInc uint32) bool {
	return h2SettingsAnomaly(ua, flags, enablePush, initialWindow, windowInc)
}

func H2PseudoOrderMismatch(ua string, order uint16, count uint8) bool {
	return h2PseudoOrderMismatch(ua, order, count)
}

func H2DowngradeArtifact(flags uint8) bool {
	return h2DowngradeArtifact(flags)
}

func FilterDeadlineExceededEvt(evt *domain.Event, ctx context.Context) bool {
	return filterDeadlineExceededEvt(evt, ctx)
}

func FilterDeadlineRemainingEvt(evt *domain.Event, ctx context.Context) (time.Duration, bool) {
	return filterDeadlineRemainingEvt(evt, ctx)
}

func EventHasFraudL3(evt *domain.Event) bool {
	return eventHasFraudL3(evt)
}

func FraudBlacklistShardIndex(ip string) uint32 {
	return fraudBlacklistShardIndex(ip)
}

type RedisShardObservability = redisShardObservability

func NewRedisShardObservability(numShards int, sampleMask uint64) RedisShardObservability {
	return newRedisShardObservability(numShards, sampleMask)
}
