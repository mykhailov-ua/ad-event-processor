package ingestion

import (
	"espx/pkg/faultproof"

	"testing"
	"time"

	"espx/internal/metrics"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func newTestUDPControl(numShards int) *UDPControl {
	return NewUDPControl(UDPControlConfig{
		Enabled:      true,
		FailClosed:   true,
		TrackerID:    1,
		SyncInterval: 100 * time.Millisecond,
		NumShards:    numShards,
		InitialRPS:   10_000,
	})
}

func encodeTestEpochPacket(t *testing.T, epoch int64, rps uint64, msgType uint8, flags uint16, numShards uint8) []byte {
	t.Helper()
	var limits UDPControlLimits
	limits.NumShards = numShards
	for i := uint8(0); i < numShards; i++ {
		limits.Limits[i] = rps
	}
	hash := ComputeUDPConfigHash(epoch, 0, &limits)
	hdr := &UDPHeader{
		CoarseTimeNs: time.Now().UnixNano(),
		EpochID:      epoch,
		ConfigHash:   hash,
		Flags:        flags,
	}
	var buf [256]byte
	n := EncodeQuotaEpochDatagram(buf[:], msgType, hdr, &limits)
	require.Greater(t, n, UDPHeaderSize)
	return append([]byte(nil), buf[:n]...)
}

func TestFault_UDP_EpochGapTighten(t *testing.T) {
	c := newTestUDPControl(2)
	require.True(t, c.ApplyPacket(encodeTestEpochPacket(t, 1, 10_000, UDPMsgQuotaEpoch, 0, 2)))
	require.Equal(t, int64(1), c.CurrentEpoch())
	require.True(t, c.ApplyPacket(encodeTestEpochPacket(t, 5, 1_000, UDPMsgQuotaEpoch, 0, 2)))
	require.Equal(t, int64(5), c.CurrentEpoch())
	require.Equal(t, uint64(1_000), c.ShardLimitRPS(0))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.UDPControlGapTightenTotal))
	faultproof.Log(t, "udp_epoch_gap_tighten", map[string]string{
		"epoch": "5", "limit_rps": "1000", "fail_closed": "true",
	})
}

func TestFault_UDP_EpochGapLoosenBlock(t *testing.T) {
	c := newTestUDPControl(2)
	require.True(t, c.ApplyPacket(encodeTestEpochPacket(t, 1, 5_000, UDPMsgQuotaEpoch, 0, 2)))
	before := testutil.ToFloat64(metrics.UDPControlLoosenBlockedTotal)
	require.False(t, c.ApplyPacket(encodeTestEpochPacket(t, 4, 20_000, UDPMsgQuotaEpoch, 0, 2)))
	require.Equal(t, int64(1), c.CurrentEpoch())
	require.Equal(t, uint64(5_000), c.ShardLimitRPS(0))
	require.Equal(t, before+1, testutil.ToFloat64(metrics.UDPControlLoosenBlockedTotal))
	faultproof.Log(t, "udp_epoch_gap_loosen_block", map[string]string{
		"blocked": "true", "epoch_unchanged": "true",
	})
}

func TestFault_UDP_EpochGapLoosenSnapshot(t *testing.T) {
	c := newTestUDPControl(2)
	require.True(t, c.ApplyPacket(encodeTestEpochPacket(t, 1, 5_000, UDPMsgQuotaEpoch, 0, 2)))
	require.True(t, c.ApplyPacket(encodeTestEpochPacket(t, 9, 25_000, UDPMsgConfigSnapshot, UDPFlagSnapshot, 2)))
	require.Equal(t, int64(9), c.CurrentEpoch())
	require.Equal(t, uint64(25_000), c.ShardLimitRPS(0))
}

func TestFault_UDP_StaleFailClosed(t *testing.T) {
	c := newTestUDPControl(2)
	c.syncInterval = 50 * time.Millisecond
	require.True(t, c.ApplyPacket(encodeTestEpochPacket(t, 1, 20_000, UDPMsgQuotaEpoch, 0, 2)))
	c.markFresh()
	c.lastPacketMono.Store(monotonicNano() - int64(200*time.Millisecond))
	c.checkStale()
	require.Equal(t, UDPChannelStale, c.ChannelState())
	floor := c.ShardLimitRPS(0)
	require.Equal(t, uint64(1000), floor)
	faultproof.Log(t, "udp_stale_fail_closed", map[string]string{
		"floor_rps": "1000", "stale": "true",
	})
}

func TestFault_UDP_LossReorder(t *testing.T) {
	c := newTestUDPControl(2)
	require.True(t, c.ApplyPacket(encodeTestEpochPacket(t, 3, 8_000, UDPMsgQuotaEpoch, 0, 2)))
	require.False(t, c.ApplyPacket(encodeTestEpochPacket(t, 2, 1_000, UDPMsgQuotaEpoch, 0, 2)))
	require.Equal(t, int64(3), c.CurrentEpoch())
	require.Equal(t, uint64(8_000), c.ShardLimitRPS(0))
	faultproof.Log(t, "udp_loss_reorder", map[string]string{
		"stale_dropped": "true", "epoch": "3",
	})
}
