package postback

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClampOutboundDelaySeconds_holdout(t *testing.T) {
	require.Equal(t, int32(0), ClampOutboundDelaySeconds(-1))
	require.Equal(t, int32(0), ClampOutboundDelaySeconds(0))
	require.Equal(t, int32(300), ClampOutboundDelaySeconds(300))
	require.Equal(t, int32(maxOutboundDelaySeconds), ClampOutboundDelaySeconds(maxOutboundDelaySeconds+1))
}

func TestPostbackNotBefore_holdout(t *testing.T) {
	base := time.Unix(1_700_000_000, 0).UTC()
	require.Equal(t, base, PostbackNotBefore(0, base))
	require.Equal(t, base.Add(90*time.Second), PostbackNotBefore(90, base))
}
