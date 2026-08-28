package settingsadmin_test

import (
	"testing"
	"time"

	"ad-event-processor/internal/settingsadmin"
)

func TestResolveBlacklistExpiry_ManualPermanent(t *testing.T) {
	cfg := settingsadmin.BlacklistTTLFromConfig(24, 168)
	expires := settingsadmin.ResolveBlacklistExpiry("manual", nil, cfg)
	if expires.Valid {
		t.Fatal("manual blocks should not expire by default")
	}
}

func TestResolveBlacklistExpiry_FraudDefault(t *testing.T) {
	cfg := settingsadmin.BlacklistTTLFromConfig(24, 168)
	expires := settingsadmin.ResolveBlacklistExpiry("fraud", nil, cfg)
	if !expires.Valid {
		t.Fatal("expected fraud TTL")
	}
	if time.Until(expires.Time) < 167*time.Hour {
		t.Fatalf("fraud TTL too short: %v", expires.Time)
	}
}

func TestResolveBlacklistExpiry_ExplicitSeconds(t *testing.T) {
	cfg := settingsadmin.BlacklistTTLFromConfig(24, 168)
	ttl := int64(3600)
	expires := settingsadmin.ResolveBlacklistExpiry("auto", &ttl, cfg)
	if !expires.Valid {
		t.Fatal("expected explicit TTL")
	}
	if time.Until(expires.Time) < 59*time.Minute {
		t.Fatalf("explicit TTL too short: %v", expires.Time)
	}
}

func TestResolveBlacklistExpiry_ZeroTTLPermanent(t *testing.T) {
	cfg := settingsadmin.BlacklistTTLFromConfig(24, 168)
	zero := int64(0)
	expires := settingsadmin.ResolveBlacklistExpiry("auto", &zero, cfg)
	if expires.Valid {
		t.Fatal("zero TTL should mean permanent")
	}
}
