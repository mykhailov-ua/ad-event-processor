package gnet

import (
	"testing"

	"ad-event-processor/internal/metrics"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestPutRequestBuffer_trimIncrementsMetric(t *testing.T) {
	before := testutil.ToFloat64(metrics.RequestBufferPoolTrimTotal)
	buf := make([]byte, maxPoolObjectSize+1)
	ptr := &buf
	putRequestBuffer(ptr)
	after := testutil.ToFloat64(metrics.RequestBufferPoolTrimTotal)
	require.Equal(t, before+1, after)
}

func TestConnContextReset_trimsOversizedBufSlice(t *testing.T) {
	before := testutil.ToFloat64(metrics.ConnContextOversizedBufferTotal)
	s := &Server{}
	ctx := newConnContext()
	ctx.BufSlice = make([]byte, connContextBufSliceCapLimit+1)
	s.resetConnContextForReuse(ctx)
	require.Equal(t, 4096, cap(ctx.BufSlice))
	after := testutil.ToFloat64(metrics.ConnContextOversizedBufferTotal)
	require.Equal(t, before+1, after)
}
