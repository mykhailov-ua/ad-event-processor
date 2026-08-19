package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCommand(t *testing.T) {
	cmd, args := parseCommand("/trial")
	require.Equal(t, "trial", cmd)
	require.Empty(t, args)

	cmd, args = parseCommand("/start@ad-event-processorBot hello")
	require.Equal(t, "start", cmd)
	require.Equal(t, "hello", args)

	cmd, _ = parseCommand("  ")
	require.Empty(t, cmd)
}
