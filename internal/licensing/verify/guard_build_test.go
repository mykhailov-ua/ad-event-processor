//go:build !license_guard

package verify_test

import (
	"testing"

	"ad-event-processor/internal/licensing/verify"

	"github.com/stretchr/testify/require"
)

func TestGuard_NotCompiledInDefaultBuild(t *testing.T) {
	require.False(t, verify.GuardCompiledIn())
}
