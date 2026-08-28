package config

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig_GnetEventLoopCount_explicit(t *testing.T) {
	cfg := &Config{GnetNumEventLoops: 4}
	require.Equal(t, 4, cfg.GnetEventLoopCount())
}

func TestConfig_GnetEventLoopCount_defaultsToGOMAXPROCS_holdout(t *testing.T) {
	prev := runtime.GOMAXPROCS(0)
	runtime.GOMAXPROCS(8)
	t.Cleanup(func() { runtime.GOMAXPROCS(prev) })

	cfg := &Config{}
	require.Equal(t, 8, cfg.GnetEventLoopCount())
}
