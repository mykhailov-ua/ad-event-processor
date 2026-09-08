package probecluster

import (
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusterID_holdoutStableForSameTuple(t *testing.T) {
	secret := []byte("probe-cluster-test-secret")
	tuple := Tuple{
		CanvasHash: [16]byte{1, 2, 3, 4},
		AudioHash:  [16]byte{5, 6, 7, 8},
		WebGLHash:  [16]byte{9, 10, 11, 12},
		TLSJA3:     "ja3-test",
		TLSJA4:     "ja4-test",
		TCPSig:     0x01020304,
		TCPSigSet:  1,
	}
	id1 := ClusterID(secret, tuple)
	id2 := ClusterID(secret, tuple)
	assert.Equal(t, id1, id2)
}

func TestClusterID_holdoutChangesWhenTupleDiffers(t *testing.T) {
	secret := []byte("probe-cluster-test-secret")
	base := Tuple{TLSJA3: "ja3-a", TLSJA4: "ja4-a", TCPSigSet: 1, TCPSig: 9}
	other := Tuple{TLSJA3: "ja3-b", TLSJA4: "ja4-a", TCPSigSet: 1, TCPSig: 9}
	require.NotEqual(t, ClusterID(secret, base), ClusterID(secret, other))
}

func TestShouldRouteSandbox_holdoutThresholds(t *testing.T) {
	p := DefaultPolicy()
	assert.False(t, ShouldRouteSandbox(State{SessionCount: 4, CampaignCount: 3, AvgProbeScore: 80}, p))
	assert.False(t, ShouldRouteSandbox(State{SessionCount: 5, CampaignCount: 2, AvgProbeScore: 80}, p))
	assert.False(t, ShouldRouteSandbox(State{SessionCount: 5, CampaignCount: 3, AvgProbeScore: 50}, p))
	assert.True(t, ShouldRouteSandbox(State{SessionCount: 5, CampaignCount: 3, AvgProbeScore: 70}, p))
}

func TestTupleFromEvent_mapsAntifraudAndTLS(t *testing.T) {
	evt := &domain.Event{
		TLSJA3:    "ja3",
		TLSJA4:    "ja4",
		TCPSig:    42,
		TCPSigSet: 1,
	}
	snap := domain.AntifraudSnapshot{}
	snap.CanvasHash[0] = 1
	snap.WebGLHash[0] = 2
	tuple := TupleFromEvent(evt, snap)
	assert.Equal(t, "ja3", tuple.TLSJA3)
	assert.Equal(t, byte(1), tuple.CanvasHash[0])
	assert.Equal(t, byte(2), tuple.WebGLHash[0])
}
