package edge

import (
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunProgram_malformedAndFlood(t *testing.T) {
	if os.Getenv("CI") == "true" && os.Geteuid() != 0 {
		t.Skip("BPF program load may require privileges in CI")
	}

	objs, cleanup, err := OpenProgram()
	if err != nil {
		t.Skipf("BPF unavailable: %v", err)
	}
	defer cleanup()

	res, err := RunProgram(objs.XdpEdgeFilter, ProgramOptions{
		MalformedIters: 100,
		FloodPackets:   200,
		Dst:            net.IPv4(10, 0, 0, 1),
		DPort:          TrackerPort,
	})
	require.NoError(t, err)
	assert.Equal(t, 100, res.MalformedIters)
	assert.Equal(t, 200, res.FloodPackets)
}
