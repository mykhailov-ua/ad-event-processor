package legal_test

import (
	"testing"
	"time"

	"espx/pkg/legal"

	"github.com/stretchr/testify/require"
)

func TestParseAcceptance(t *testing.T) {
	raw, err := legal.MarshalAcceptance(legal.Acceptance{
		Version:    legal.Version,
		AcceptedAt: time.Now().UTC(),
		AcceptedBy: "install",
	})
	require.NoError(t, err)

	acc, err := legal.ParseAcceptance(raw)
	require.NoError(t, err)
	require.True(t, legal.IsCurrent(acc))
}

func TestParseAcceptance_empty(t *testing.T) {
	_, err := legal.ParseAcceptance("")
	require.Error(t, err)
}

func TestEulaTextEmbedded(t *testing.T) {
	require.Contains(t, legal.Text, "License grant")
}
