package ingestion

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleNodeWeights() []UDPNodeWeight {
	return []UDPNodeWeight{
		{NodeID: "tracker-1", Role: "tracker", Weight: 0.35, Score: 0.91, Provenance: ProvenanceToUDPCode("own_window")},
		{NodeID: "tracker-2", Role: "tracker", Weight: 0.65, Score: 0.78, Provenance: ProvenanceToUDPCode("own_window")},
	}
}

func encodeTestEpochWithWeights(t *testing.T, epoch int64, rps uint64, weights []UDPNodeWeight, msgType uint8, flags uint16, numShards uint8) []byte {
	t.Helper()
	var limits UDPControlLimits
	limits.NumShards = numShards
	for i := uint8(0); i < numShards; i++ {
		limits.Limits[i] = rps
	}
	hash := ComputeUDPConfigHashWithWeights(epoch, 0, &limits, weights)
	hdr := &UDPHeader{
		CoarseTimeNs: time.Now().UnixNano(),
		EpochID:      epoch,
		ConfigHash:   hash,
		Flags:        flags,
	}
	var buf [4096]byte
	n := EncodeQuotaEpochDatagramWithWeights(buf[:], msgType, hdr, &limits, weights)
	require.Greater(t, n, UDPHeaderSize)
	return append([]byte(nil), buf[:n]...)
}

func TestUDPNodeWeights_RoundTrip(t *testing.T) {
	weights := sampleNodeWeights()
	var buf [256]byte
	n := udpEncodeNodeWeights(buf[:], weights)
	require.Greater(t, n, 0)

	var decoded []UDPNodeWeight
	require.True(t, udpDecodeNodeWeights(buf[:n], &decoded))
	require.Len(t, decoded, 2)
	assert.Equal(t, "tracker-1", decoded[0].NodeID)
	assert.InDelta(t, 0.35, decoded[0].Weight, 1e-9)
	assert.InDelta(t, 0.91, decoded[0].Score, 1e-9)
	assert.Equal(t, ProvenanceToUDPCode("own_window"), decoded[0].Provenance)
}

func TestUDPNodeWeights_DatagramRoundTrip(t *testing.T) {
	c := newTestUDPControl(2)
	weights := sampleNodeWeights()
	pkt := encodeTestEpochWithWeights(t, 1, 10_000, weights, UDPMsgQuotaEpoch, 0, 2)
	require.True(t, c.ApplyPacket(pkt))
	require.Equal(t, int64(1), c.CurrentEpoch())

	got := c.NodeWeights()
	require.Len(t, got, 2)
	assert.InDelta(t, 0.35, got[0].Weight, 1e-9)
	assert.InDelta(t, 0.65, got[1].Weight, 1e-9)
}

func TestUDPNodeWeights_V1BackwardCompatible(t *testing.T) {
	c := newTestUDPControl(2)
	require.True(t, c.ApplyPacket(encodeTestEpochPacket(t, 1, 10_000, UDPMsgQuotaEpoch, 0, 2)))
	assert.Nil(t, c.NodeWeights())
	assert.False(t, c.DrainFrozen())
}

func TestUDPNodeWeights_StaleChannelEqualizes(t *testing.T) {
	c := newTestUDPControl(2)
	c.syncInterval = 50 * time.Millisecond
	weights := sampleNodeWeights()
	require.True(t, c.ApplyPacket(encodeTestEpochWithWeights(t, 1, 10_000, weights, UDPMsgQuotaEpoch, 0, 2)))
	c.markFresh()
	c.lastPacketMono.Store(monotonicNano() - int64(200*time.Millisecond))
	c.checkStale()
	require.Equal(t, UDPChannelStale, c.ChannelState())
	require.True(t, c.DrainFrozen())

	got := c.NodeWeights()
	require.Len(t, got, 2)
	assert.InDelta(t, 0.5, got[0].Weight, 1e-9)
	assert.InDelta(t, 0.5, got[1].Weight, 1e-9)
}

func TestUDPNodeWeights_EpochLagEqualizes(t *testing.T) {
	c := newTestUDPControl(2)
	weights := sampleNodeWeights()
	require.True(t, c.ApplyPacket(encodeTestEpochWithWeights(t, 1, 10_000, weights, UDPMsgQuotaEpoch, 0, 2)))
	c.lastPublisherEpoch.Store(5)

	require.True(t, c.DrainFrozen())
	got := c.NodeWeights()
	require.Len(t, got, 2)
	assert.InDelta(t, 0.5, got[0].Weight, 1e-9)
	assert.InDelta(t, 0.5, got[1].Weight, 1e-9)
}

func TestMarshalEpochPayload_IncludesNodeWeights(t *testing.T) {
	var limits UDPControlLimits
	limits.NumShards = 2
	limits.Limits[0] = 50_000
	limits.Limits[1] = 50_000
	raw, err := MarshalEpochPayload(3, &limits, sampleNodeWeights())
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"node_weights"`)
	assert.Contains(t, string(raw), `"tracker-1"`)
}

func TestEqualizeNodeWeights_sumsToOne(t *testing.T) {
	out := EqualizeNodeWeights(sampleNodeWeights())
	var sum float64
	for _, w := range out {
		sum += w.Weight
	}
	assert.InDelta(t, 1.0, sum, 1e-9)
}
