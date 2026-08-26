//go:build !license_guard

package licensing_test

import (
	"testing"

	"ad-event-processor/internal/licensing"

	"github.com/stretchr/testify/require"
)

func TestGuard_NotCompiledInDefaultBuild(t *testing.T) {
	require.False(t, licensing.GuardCompiledIn())
}
