package main

import (
	"bytes"
	"flag"
	"testing"
	"time"

	"ad-event-processor/internal/trialregistry"

	"github.com/stretchr/testify/require"
)

func TestRunListPending(t *testing.T) {
	regPath := t.TempDir() + "/trial.json"
	t.Setenv(trialregistry.EnvRegistryPath, regPath)

	reg := trialregistry.New(regPath, 0)
	_, err := reg.EnqueuePending(trialregistry.EnqueuePendingInput{
		TelegramID:       "7007",
		TelegramUsername: "pending_user",
	})
	require.NoError(t, err)

	code := runListPending([]string{"--trial-registry", regPath})
	require.Equal(t, 0, code)
}

func TestRunRejectPending(t *testing.T) {
	regPath := t.TempDir() + "/trial.json"
	reg := trialregistry.New(regPath, 0)
	req, err := reg.EnqueuePending(trialregistry.EnqueuePendingInput{TelegramID: "8008"})
	require.NoError(t, err)

	code := runRejectPending([]string{"--id", req.ID, "--reason", "test", "--trial-registry", regPath})
	require.Equal(t, 0, code)

	open, err := reg.ListPending()
	require.NoError(t, err)
	require.Empty(t, open)
}

func TestRunExpireStale(t *testing.T) {
	regPath := t.TempDir() + "/trial.json"
	t.Setenv(trialregistry.EnvRegistryPath, regPath)

	reg := trialregistry.New(regPath, 0)
	dep := "550e8400-e29b-41d4-a716-446655440000"
	past := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, reg.RecordPilotIssue(trialregistry.RecordInput{
		TelegramID:   "6006",
		DeploymentID: dep,
		LicenseKey:   dep,
		ValidUntil:   past,
	}))

	code := runExpireStale([]string{"--at", "2026-08-18T00:00:00Z", "--trial-registry", regPath})
	require.Equal(t, 0, code)

	err := reg.CheckPilotEligible(trialregistry.CheckInput{TelegramID: "6006"})
	require.ErrorIs(t, err, trialregistry.ErrTrialTelegramUsed)
}

func TestRunExpireStale_invalidAt(t *testing.T) {
	code := runExpireStale([]string{"--at", "not-a-time"})
	require.Equal(t, 1, code)
}

func TestRunExpireStale_flagError(t *testing.T) {
	fs := flag.NewFlagSet("expire-stale", flag.ContinueOnError)
	fs.SetOutput(&bytes.Buffer{})
	require.Error(t, fs.Parse([]string{"-unknown"}))
}
