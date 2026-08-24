package domain

const BlacklistUpdateChannel = "blacklist:update"

func DefaultBlacklistUpdateChannel(channel string) string {
	if channel != "" {
		return channel
	}
	return BlacklistUpdateChannel
}
