package logpipeline

import (
	"bytes"
	"testing"
	"time"

	"ad-event-processor/internal/ingestion/pb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregateFilterRejectSlices_holdoutGroupsByHourKindPlacementCountry(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 8, 24, 15, 30, 0, 0, time.UTC).Unix()
	var plain bytes.Buffer
	for range 3 {
		plain.Write(encodeRecord(t, &pb.AdStreamEvent{
			EventType:     []byte(filterRejectSampleEventType),
			CreatedAtUnix: ts,
			Payload:       []byte(`{"k":"geo","p":"zone_1","c":"DE"}`),
		}))
	}
	plain.Write(encodeRecord(t, &pb.AdStreamEvent{
		EventType:     []byte(filterRejectSampleEventType),
		CreatedAtUnix: ts,
		Payload:       []byte(`{"k":"geo","p":"zone_2","c":"US"}`),
	}))
	plain.Write(encodeRecord(t, &pb.AdStreamEvent{
		EventType:     []byte("impression"),
		CreatedAtUnix: ts,
	}))

	rows, err := aggregateFilterRejectSlices(bytes.NewReader(plain.Bytes()))
	require.NoError(t, err)
	require.Len(t, rows, 2)

	hour := time.Unix(ts, 0).UTC().Truncate(time.Hour)
	assert.Equal(t, hour, rows[0].RollupHour)
	assert.Equal(t, "geo", rows[0].RejectKind)
	assert.Equal(t, "DE", rows[0].Country)
	assert.Equal(t, "zone_1", rows[0].PlacementID)
	assert.Equal(t, uint64(3), rows[0].RejectCount)

	assert.Equal(t, "US", rows[1].Country)
	assert.Equal(t, uint64(1), rows[1].RejectCount)
}

func TestAggregateWarmAndRejectSlices_returnsBothKinds(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC).Unix()
	campaignID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	var plain bytes.Buffer
	plain.Write(encodeRecord(t, &pb.AdStreamEvent{
		EventType:     []byte("click"),
		ClickId:       []byte("click-1"),
		CampaignId:    campaignID[:],
		CreatedAtUnix: ts,
	}))
	plain.Write(encodeRecord(t, &pb.AdStreamEvent{
		EventType:     []byte(filterRejectSampleEventType),
		CreatedAtUnix: ts,
		Payload:       []byte(`{"k":"budget","p":"z","c":"US"}`),
	}))

	rollups, slices, err := aggregateWarmAndRejectSlices(plain.Bytes(), "seg", "sha")
	require.NoError(t, err)
	require.NotEmpty(t, rollups)
	require.Len(t, slices, 1)
	assert.Equal(t, "budget", slices[0].RejectKind)
}
