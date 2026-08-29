package filter

import (
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestVPPFilter_skipsNonVPP(t *testing.T) {
	campID := uuid.New()
	reg := NewRegistry(nil)
	reg.Add(campID, uuid.New(), nil, "", domain.PacingModeAsap, 0, "UTC", 0, 0, nil)

	sw := &SettingsWatcher{}
	sw.vppRatios.Store(&VPPRatioSnapshot{Ratios: map[uuid.UUID]float32{campID: 0.1}})

	f := NewVPPFilter(reg, sw)
	err := f.Check(t.Context(), &domain.Event{CampaignID: campID})
	require.NoError(t, err)
}

func TestVPPFilter_throttlesWhenRatioLow(t *testing.T) {
	campID := uuid.New()
	reg := NewRegistry(nil)
	reg.Add(campID, uuid.New(), nil, "", domain.PacingModeVpp, 0, "UTC", 0, 0, nil)

	sw := &SettingsWatcher{}
	sw.vppRatios.Store(&VPPRatioSnapshot{Ratios: map[uuid.UUID]float32{campID: 0.0}})

	f := NewVPPFilter(reg, sw)
	err := f.Check(t.Context(), &domain.Event{CampaignID: campID})
	require.ErrorIs(t, err, ErrPacingExhausted)
}

func BenchmarkPacingRead(b *testing.B) {
	campID := uuid.New()
	sw := &SettingsWatcher{}
	sw.vppRatios.Store(&VPPRatioSnapshot{Ratios: map[uuid.UUID]float32{campID: 0.75}})

	b.ReportAllocs()
	for b.Loop() {
		_ = sw.GetVPPRatio(campID)
	}
}

func BenchmarkVPPAllow(b *testing.B) {
	campID := uuid.New()
	b.ReportAllocs()
	benchN := 0
	for b.Loop() {
		_ = vppAllow(campID, 0.75, int64(benchN))
		benchN++
	}
}
