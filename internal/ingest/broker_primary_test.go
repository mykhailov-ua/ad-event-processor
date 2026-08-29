package ingest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSetBrokerProducer_defersLuaStreamWrite(t *testing.T) {
	uf := NewUnifiedFilter(nil, nil, nil, nil, 0, 0, 0, 0, 0, 0, "ad:events:stream", 0)
	fe := NewFilterEngine(time.Second, uf)
	h := &AdsPacketHandler{filterEngine: fe}
	bp, err := NewBrokerProducer(DefaultBrokerProducerConfig())
	require.NoError(t, err)
	defer func() { _ = bp.Close() }()

	h.SetBrokerProducer(bp)
	require.Equal(t, "fcap:ignored", uf.streamKeyVal.s)
}
