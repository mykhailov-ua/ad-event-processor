package monotime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNano_holdoutMonotonic(t *testing.T) {
	start := Nano()
	time.Sleep(time.Millisecond)
	end := Nano()
	require.Greater(t, end, start)
}
