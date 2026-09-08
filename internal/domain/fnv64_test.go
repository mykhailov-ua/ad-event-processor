package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFNV64String_stable(t *testing.T) {
	require.Equal(t, FNV64String("fcap:c:test:u:"), FNV64String("fcap:c:test:u:"))
	require.NotZero(t, FNV64String("x"))
	require.Zero(t, FNV64String(""))
}
