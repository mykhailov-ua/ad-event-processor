package filter

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/crowdprobe"
	"ad-event-processor/pkg/probecluster"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProbeClusterStore_holdoutSameTupleIncrementsSession(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewProbeClusterStore(client, 30*24*time.Hour)
	secret := []byte("probe-cluster-integration-secret")
	tuple := probecluster.Tuple{
		TLSJA3:    "ja3-integration",
		TLSJA4:    "ja4-integration",
		TCPSig:    0xdeadbeef,
		TCPSigSet: 1,
	}
	clusterID := probecluster.ClusterID(secret, tuple)
	campaignA := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	campaignB := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	campaignC := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		state, err := store.Observe(ctx, ProbeClusterObserveInput{
			ClusterID:  clusterID,
			SessionID:  fmt.Sprintf("session-%d", i),
			CampaignID: campaignA,
			ProbeScore: 80,
			JA3:        tuple.TLSJA3,
			JA4:        tuple.TLSJA4,
			TCPSig:     tuple.TCPSig,
			TCPSigSet:  tuple.TCPSigSet,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(i+1), state.SessionCount)
	}
	state, err := store.Observe(ctx, ProbeClusterObserveInput{
		ClusterID:  clusterID,
		SessionID:  "session-e",
		CampaignID: campaignB,
		ProbeScore: 80,
		JA3:        tuple.TLSJA3,
		JA4:        tuple.TLSJA4,
		TCPSig:     tuple.TCPSig,
		TCPSigSet:  tuple.TCPSigSet,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(5), state.SessionCount)
	assert.Equal(t, int64(2), state.CampaignCount)

	state, err = store.Observe(ctx, ProbeClusterObserveInput{
		ClusterID:  clusterID,
		SessionID:  "session-f",
		CampaignID: campaignC,
		ProbeScore: 80,
		JA3:        tuple.TLSJA3,
		JA4:        tuple.TLSJA4,
		TCPSig:     tuple.TCPSig,
		TCPSigSet:  tuple.TCPSigSet,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(6), state.SessionCount)
	assert.Equal(t, int64(3), state.CampaignCount)
	assert.GreaterOrEqual(t, state.AvgProbeScore, uint8(65))
	assert.True(t, probecluster.ShouldRouteSandbox(state, probecluster.Policy{
		MinSessions:    5,
		MinCampaigns:   3,
		ScoreThreshold: 65,
	}))
}

func TestProbeClusterFilter_holdoutRoutesHotCluster(t *testing.T) {
	before := testutil.ToFloat64(metrics.ProbeClusterRouteTotal)
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewProbeClusterStore(client, 30*24*time.Hour)
	secret := []byte("probe-cluster-filter-secret")
	campaignA := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	campaignB := uuid.MustParse("bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb")
	reg := &crowdProbeCampRegistry{camp: &domain.Campaign{
		ID:                 campaignA,
		SafePageEnabled:    true,
		AttestationEnabled: true,
	}}
	f := NewProbeClusterFilter(reg)
	f.SetEnabled(true)
	f.SetSecret(secret)
	f.SetStore(store)
	f.SetPolicy(probecluster.Policy{MinSessions: 2, MinCampaigns: 2, ScoreThreshold: crowdprobe.BehaviorSignalThreshold})
	ctx := context.Background()
	tupleJA3 := "ja3-filter"
	evt1 := &domain.Event{
		ClickID:            "sess-1",
		CampaignID:         campaignA,
		Type:               "click",
		TLSJA3:             tupleJA3,
		TLSJA4:             "ja4-filter",
		TCPSig:             99,
		TCPSigSet:          1,
		ProbeBehaviorScore: 70,
		ProbeFeaturesSet:   1,
	}
	require.NoError(t, f.Check(ctx, evt1))
	assert.Equal(t, uint8(0), evt1.ProbeClusterRoute)
	evt2 := &domain.Event{
		ClickID:            "sess-2",
		CampaignID:         campaignB,
		Type:               "click",
		TLSJA3:             tupleJA3,
		TLSJA4:             "ja4-filter",
		TCPSig:             99,
		TCPSigSet:          1,
		ProbeBehaviorScore: 70,
		ProbeFeaturesSet:   1,
	}
	require.NoError(t, f.Check(ctx, evt2))
	assert.Equal(t, uint8(1), evt2.ProbeClusterRoute)
	assert.Equal(t, uint8(1), evt2.ProbeClusterSet)
	assert.GreaterOrEqual(t, testutil.ToFloat64(metrics.ProbeClusterRouteTotal), before+1)
}

func TestProbeClusterStore_GetSummaryBatch_holdoutMatchesGetSummary(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewProbeClusterStore(client, 30*24*time.Hour)
	secret := []byte("probe-cluster-batch-secret")
	tuple := probecluster.Tuple{
		TLSJA3:    "ja3-batch",
		TLSJA4:    "ja4-batch",
		TCPSig:    0xabc123,
		TCPSigSet: 1,
	}
	clusterID := probecluster.ClusterID(secret, tuple)
	clusterHex := probecluster.ClusterIDHex(clusterID)
	campaignID := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	ctx := context.Background()
	_, err := store.Observe(ctx, ProbeClusterObserveInput{
		ClusterID:  clusterID,
		SessionID:  "session-batch",
		CampaignID: campaignID,
		ProbeScore: 70,
		JA3:        tuple.TLSJA3,
		JA4:        tuple.TLSJA4,
		TCPSig:     tuple.TCPSig,
		TCPSigSet:  tuple.TCPSigSet,
	})
	require.NoError(t, err)

	batch, err := store.GetSummaryBatch(ctx, []string{clusterHex, "00000000000000000000000000000000"})
	require.NoError(t, err)
	require.Len(t, batch, 1)

	single, ok, err := store.GetSummary(ctx, clusterHex)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, single, batch[clusterHex])

	require.NoError(t, store.MarkExportedBatch(ctx, []string{clusterHex}))
	single, ok, err = store.GetSummary(ctx, clusterHex)
	require.NoError(t, err)
	require.True(t, ok)
	assert.True(t, single.ExportDone)
}
