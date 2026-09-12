package postback

import "time"

const maxOutboundDelaySeconds = 7 * 24 * 60 * 60

func ClampOutboundDelaySeconds(seconds int32) int32 {
	if seconds <= 0 {
		return 0
	}
	if seconds > maxOutboundDelaySeconds {
		return maxOutboundDelaySeconds
	}
	return seconds
}

func PostbackNotBefore(delaySeconds int32, base time.Time) time.Time {
	delaySeconds = ClampOutboundDelaySeconds(delaySeconds)
	if delaySeconds <= 0 {
		return base
	}
	return base.Add(time.Duration(delaySeconds) * time.Second)
}
