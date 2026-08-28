package settingsadmin

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type blacklistTTLConfig struct {
	autoTTLHours  int
	fraudTTLHours int
}

type BlacklistTTLConfig struct {
	AutoTTLHours  int
	FraudTTLHours int
}

func BlacklistTTLFromConfig(autoTTLHours, fraudTTLHours int) BlacklistTTLConfig {
	out := BlacklistTTLConfig{
		AutoTTLHours:  24,
		FraudTTLHours: 168,
	}
	if autoTTLHours > 0 {
		out.AutoTTLHours = autoTTLHours
	}
	if fraudTTLHours > 0 {
		out.FraudTTLHours = fraudTTLHours
	}
	return out
}

func blacklistTTLFromHost(host Host) blacklistTTLConfig {
	out := blacklistTTLConfig{
		autoTTLHours:  24,
		fraudTTLHours: 168,
	}
	if host == nil {
		return out
	}
	if h := host.BlacklistAutoTTLHours(); h > 0 {
		out.autoTTLHours = h
	}
	if h := host.BlacklistFraudTTLHours(); h > 0 {
		out.fraudTTLHours = h
	}
	return out
}

func normalizeBlacklistReason(reason string) string {
	if reason == "" {
		return "manual"
	}
	return reason
}

func NormalizeBlacklistReason(reason string) string {
	return normalizeBlacklistReason(reason)
}

func ResolveBlacklistExpiry(reason string, ttlSeconds *int64, cfg BlacklistTTLConfig) pgtype.Timestamptz {
	return resolveBlacklistExpiry(reason, ttlSeconds, blacklistTTLConfig{
		autoTTLHours:  cfg.AutoTTLHours,
		fraudTTLHours: cfg.FraudTTLHours,
	})
}

func resolveBlacklistExpiry(reason string, ttlSeconds *int64, cfg blacklistTTLConfig) pgtype.Timestamptz {
	if ttlSeconds != nil {
		if *ttlSeconds <= 0 {
			return pgtype.Timestamptz{}
		}
		return pgtype.Timestamptz{
			Time:  time.Now().UTC().Add(time.Duration(*ttlSeconds) * time.Second),
			Valid: true,
		}
	}

	reason = normalizeBlacklistReason(reason)
	switch reason {
	case "manual":
		return pgtype.Timestamptz{}
	case "auto":
		if cfg.autoTTLHours <= 0 {
			return pgtype.Timestamptz{}
		}
		return pgtype.Timestamptz{
			Time:  time.Now().UTC().Add(time.Duration(cfg.autoTTLHours) * time.Hour),
			Valid: true,
		}
	case "fraud":
		if cfg.fraudTTLHours <= 0 {
			return pgtype.Timestamptz{}
		}
		return pgtype.Timestamptz{
			Time:  time.Now().UTC().Add(time.Duration(cfg.fraudTTLHours) * time.Hour),
			Valid: true,
		}
	default:
		return pgtype.Timestamptz{}
	}
}
