package ingest

import (
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestStreamProducerAdmissionRaceWithoutReserve(t *testing.T) {
	const queueCapLimit = 32
	const admissionPct = 75

	p := NewStreamProducerQueueForTest(queueCapLimit, 23)

	cfg := &config.Config{StreamProducerAdmissionPct: admissionPct}
	sharder := NewJumpHashSharder(1)
	producers := []*StreamProducer{p}
	campaignID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	const workers = 16
	_, reject := rejectIfStreamProducerOverloaded(cfg, sharder, producers, nil, campaignID)
	require.False(t, reject, "read-only admission passes at 71%% fill before concurrent arrivals")

	queueFull := 0
	for range workers {
		evt := &domain.Event{
			ClickID:    "race",
			CampaignID: campaignID,
			Type:       "click",
		}
		if err := p.Process(evt); err != nil {
			queueFull++
		}
	}

	require.Equal(t, 7, queueFull, "without reservation 7 of 16 enqueues exceed remaining channel capacity")
}

func TestStreamProducerReservePreventsQueueFull(t *testing.T) {
	const queueCapLimit = 32
	const admissionPct = 75

	p := NewStreamProducerQueueForTest(queueCapLimit, 20)

	cfg := &config.Config{StreamProducerAdmissionPct: admissionPct}
	sharder := NewJumpHashSharder(1)
	producers := []*StreamProducer{p}
	campaignID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	const workers = 16
	var leases []streamAdmissionLease
	for range workers {
		lease, _, ok := tryAcquireStreamAdmission(cfg, sharder, producers, nil, campaignID, false)
		if ok {
			leases = append(leases, lease)
		}
	}
	require.Len(t, leases, 4, "only headroom slots should be reserved")

	queueFull := 0
	for _, lease := range leases {
		evt := &domain.Event{
			ClickID:    "reserved",
			CampaignID: campaignID,
			Type:       "click",
		}
		if err := p.ProcessReserved(evt); err != nil {
			queueFull++
		}
		lease.Clear()
	}
	require.Zero(t, queueFull, "reserved enqueue must not return ErrQueueFull")
}

func TestStreamProducerAdmissionReject(t *testing.T) {
	p := NewStreamProducerQueueForTest(4, 4)

	cfg := &config.Config{StreamProducerAdmissionPct: 75}
	sharder := NewJumpHashSharder(1)
	producers := []*StreamProducer{p}
	campaignID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	_, kind, acquired := tryAcquireStreamAdmission(cfg, sharder, producers, nil, campaignID, false)
	require.False(t, acquired)
	require.Equal(t, filterRejectProducerOverload, kind)
	require.Equal(t, 100, p.QueuePressurePct())
}

func TestStreamProducerAdmissionAllowsHeadroom(t *testing.T) {
	p := NewStreamProducerQueueForTest(100, 50)

	cfg := &config.Config{StreamProducerAdmissionPct: 85}
	sharder := NewJumpHashSharder(1)
	producers := []*StreamProducer{p}
	campaignID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	_, _, acquired := tryAcquireStreamAdmission(cfg, sharder, producers, nil, campaignID, false)
	require.True(t, acquired)
}
