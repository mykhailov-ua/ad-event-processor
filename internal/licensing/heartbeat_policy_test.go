package licensing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDetermineEffectiveState_transitions(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	claims := &LicenseClaims{
		ValidFrom:  now.Add(-48 * time.Hour),
		ValidUntil: now.Add(48 * time.Hour),
		GraceDays:  7,
	}
	policy := HeartbeatPolicy{OfflineGraceDays: 14, RenewBeforeDays: 7}

	tests := []struct {
		name             string
		offlineSince     time.Time
		heartbeatOffline bool
		want             LicenseState
	}{
		{
			name:             "active heartbeat ok",
			heartbeatOffline: false,
			want:             StateActive,
		},
		{
			name:             "offline warn day zero",
			offlineSince:     now,
			heartbeatOffline: true,
			want:             StateOfflineWarn,
		},
		{
			name:             "offline warn mid period",
			offlineSince:     now.Add(-6 * 24 * time.Hour),
			heartbeatOffline: true,
			want:             StateOfflineWarn,
		},
		{
			name:             "offline grace tail",
			offlineSince:     now.Add(-10 * 24 * time.Hour),
			heartbeatOffline: true,
			want:             StateOfflineGrace,
		},
		{
			name:             "offline expired",
			offlineSince:     now.Add(-15 * 24 * time.Hour),
			heartbeatOffline: true,
			want:             StateExpired,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DetermineEffectiveState(claims, now, false, tc.offlineSince, tc.heartbeatOffline, policy)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDetermineEffectiveState_jwtGraceIndependentOfHeartbeat(t *testing.T) {
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	claims := &LicenseClaims{
		ValidFrom:  now.Add(-96 * time.Hour),
		ValidUntil: now.Add(-24 * time.Hour),
		GraceDays:  7,
	}
	policy := HeartbeatPolicy{OfflineGraceDays: 14, RenewBeforeDays: 7}
	offlineSince := now.Add(-20 * 24 * time.Hour)
	got := DetermineEffectiveState(claims, now, false, offlineSince, true, policy)
	assert.Equal(t, StateGrace, got)
}

func TestDetermineEffectiveState_revokedAndJwtExpired(t *testing.T) {
	now := time.Now()
	claims := &LicenseClaims{
		ValidFrom:  now.Add(-96 * time.Hour),
		ValidUntil: now.Add(-48 * time.Hour),
		GraceDays:  1,
	}
	policy := HeartbeatPolicy{OfflineGraceDays: 14, RenewBeforeDays: 7}
	offlineSince := now.Add(-2 * 24 * time.Hour)

	assert.Equal(t, StateRevoked, DetermineEffectiveState(claims, now, true, offlineSince, true, policy))
	assert.Equal(t, StateExpired, DetermineEffectiveState(claims, now, false, offlineSince, true, policy))
}

func TestIngestAllowed(t *testing.T) {
	assert.True(t, IngestAllowed(StateActive))
	assert.True(t, IngestAllowed(StateOfflineWarn))
	assert.True(t, IngestAllowed(StateOfflineGrace))
	assert.True(t, IngestAllowed(StateGrace))
	assert.False(t, IngestAllowed(StateExpired))
	assert.False(t, IngestAllowed(StateRevoked))
}

func TestBannerSeverity(t *testing.T) {
	assert.Equal(t, "warn", BannerSeverity(StateOfflineWarn))
	assert.Equal(t, "grace", BannerSeverity(StateOfflineGrace))
	assert.Equal(t, "", BannerSeverity(StateActive))
}
