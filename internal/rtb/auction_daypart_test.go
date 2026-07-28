package rtb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDaypartMaskFromHours(t *testing.T) {
	tests := []struct {
		name  string
		hours map[int16]struct{}
		want  uint32
	}{
		{name: "empty allows all", hours: nil, want: 0},
		{name: "single hour", hours: map[int16]struct{}{9: {}}, want: 1 << 9},
		{name: "multiple hours", hours: map[int16]struct{}{0: {}, 12: {}, 23: {}}, want: (1 << 0) | (1 << 12) | (1 << 23)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, DaypartMaskFromHours(tc.hours))
		})
	}
}

func TestScheduleOpen_table(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	_, offset := time.Date(2026, 7, 15, 12, 0, 0, 0, loc).Zone()

	tests := []struct {
		name        string
		start       int64
		end         int64
		mask        uint32
		tzOffset    int32
		now         time.Time
		wantAllowed bool
	}{
		{
			name: "in window no daypart", now: time.Date(2026, 7, 15, 15, 0, 0, 0, time.UTC),
			start:       time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC).Unix(),
			end:         time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC).Unix(),
			wantAllowed: true,
		},
		{
			name: "before start", now: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			start:       time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC).Unix(),
			wantAllowed: false,
		},
		{
			name: "after end", now: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
			end:         time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC).Unix(),
			wantAllowed: false,
		},
		{
			name: "daypart in hour", now: time.Date(2026, 7, 15, 14, 0, 0, 0, time.UTC),
			mask: 1 << 14, wantAllowed: true,
		},
		{
			name: "daypart out of hour", now: time.Date(2026, 7, 15, 3, 0, 0, 0, time.UTC),
			mask: 1 << 9, wantAllowed: false,
		},
		{
			name: "tz offset local hour 9 allowed", now: time.Date(2026, 7, 15, 13, 0, 0, 0, time.UTC),
			mask: 1 << 9, tzOffset: int32(offset), wantAllowed: true,
		},
		{
			name: "missing daypart mask fail-open", now: time.Date(2026, 7, 15, 3, 0, 0, 0, time.UTC),
			mask: 0, wantAllowed: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := scheduleOpen(tc.start, tc.end, tc.mask, tc.tzOffset, tc.now.Unix())
			assert.Equal(t, tc.wantAllowed, got)
		})
	}
}

func TestAuction_daypartRejectBeforeScan(t *testing.T) {
	SetMetricsEnabled(false)
	store := NewBudgetStore()
	reg := NewRegistry(store)

	openHour := time.Now().UTC().Hour()
	closedMask := uint32(0)
	if openHour != 9 {
		closedMask = 1 << 9
	} else {
		closedMask = 1 << 10
	}

	reg.UpdateCampaigns([]CampaignData{{
		ID: CampaignID(1), Bid: 200, DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7,
		Weight: 1, Budget: 5000, DaypartMask: closedMask,
	}})

	_, reason := reg.RunAuction(&BidRequest{
		DeviceType: 1, CategoryMask: 1, GeoHash: 7, MinBid: 50,
		NowUnix: time.Now().UTC().Unix(),
	})
	assert.Equal(t, NoBidDaypartClosed, reason)
}

func TestAuction_daypartInWindowClears(t *testing.T) {
	SetMetricsEnabled(false)
	store := NewBudgetStore()
	reg := NewRegistry(store)
	hour := time.Now().UTC().Hour()

	reg.UpdateCampaigns([]CampaignData{{
		ID: CampaignID(1), Bid: 200, DeviceMask: 1, CategoryMask: 1, GeoHashVal: 7,
		Weight: 1, Budget: 5000, DaypartMask: 1 << uint(hour),
	}})

	res, reason := reg.RunAuction(&BidRequest{
		DeviceType: 1, CategoryMask: 1, GeoHash: 7, MinBid: 50,
		NowUnix: time.Now().UTC().Unix(),
	})
	require.Equal(t, NoBidNone, reason)
	assert.Equal(t, CampaignID(1), res.CampaignID)
}
