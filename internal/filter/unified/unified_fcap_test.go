package unified

import (
	"testing"

	"ad-event-processor/internal/domain"
	filt "ad-event-processor/internal/filter"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUnifiedFilter_checkFreqLimitGo_localLedger_holdout(t *testing.T) {
	f := NewUnifiedFilter(nil, nil, &filt.MockRegistry{}, nil, 0, 0, 0, 0, 0, 0, "", 0)
	camp := &domain.Campaign{
		PacingMode:    domain.PacingModeAsap,
		FreqLimit:     2,
		FcapKeyPrefix: "fcap:c:123:u:",
	}
	click := &domain.Event{Type: "click", CampaignID: uuid.New(), UserID: "u1"}

	exceeded, err := f.CheckFreqLimitGo(click, camp)
	require.NoError(t, err)
	require.False(t, exceeded)

	lookup := fcapLookupKey(click, camp)
	require.NotZero(t, lookup)
	require.True(t, f.localFcapLedger.TryAcquire(lookup, 2, 3600, 1_700_000_000))
	require.True(t, f.localFcapLedger.TryAcquire(lookup, 2, 3600, 1_700_000_000))

	exceeded, err = f.CheckFreqLimitGo(click, camp)
	require.ErrorIs(t, err, filt.ErrFreqLimitExceeded)
	require.True(t, exceeded)
}

func TestUnifiedFilter_needsFullLuaPath_fcapWithoutSettingsWatcher(t *testing.T) {
	f := NewUnifiedFilter(nil, nil, &filt.MockRegistry{}, nil, 0, 0, 0, 0, 0, 0, "", 0)
	f.SetLuaFastPathEnabled(true)
	camp := &domain.Campaign{PacingMode: domain.PacingModeAsap, FreqLimit: 3, FcapKeyPrefix: "fcap:c:x:u:"}
	evt := &domain.Event{Type: "click", UserID: "u", CampaignID: uuid.New()}
	require.False(t, f.NeedsFullLuaPath(evt, camp))
}

func TestUnifiedFilter_rollbackLocalFcapForEvent_holdout(t *testing.T) {
	f := NewUnifiedFilter(nil, nil, &filt.MockRegistry{}, nil, 0, 0, 0, 0, 0, 0, "", 0)
	camp := &domain.Campaign{
		PacingMode:    domain.PacingModeAsap,
		FreqLimit:     2,
		FcapKeyPrefix: "fcap:c:123:u:",
	}
	evt := &domain.Event{Type: "click", UserID: "u1", ClickID: "c1"}
	lookup, err := f.tryAcquireLocalFcap(evt, camp)
	require.NoError(t, err)
	require.Equal(t, lookup, evt.LocalFcapLookup)

	f.rollbackLocalFcapForEvent(evt)
	require.Zero(t, evt.LocalFcapLookup)
	_, err = f.tryAcquireLocalFcap(evt, camp)
	require.NoError(t, err)
}
