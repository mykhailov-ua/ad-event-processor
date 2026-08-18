package trialregistry

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCheckPilotEligible_telegram(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	cooldown := 60 * 24 * time.Hour

	err := checkPilotEligible(now, cooldown, nil, CheckInput{TelegramID: "111"})
	require.NoError(t, err)

	anchors := []AnchorRecord{{
		AnchorType:  AnchorTelegram,
		AnchorValue: "111",
		Status:      StatusActive,
		IssuedAt:    now.Add(-24 * time.Hour),
	}}
	err = checkPilotEligible(now, cooldown, anchors, CheckInput{TelegramID: "111"})
	require.ErrorIs(t, err, ErrTrialTelegramUsed)

	anchors[0].Status = StatusRevoked
	err = checkPilotEligible(now, cooldown, anchors, CheckInput{TelegramID: "111"})
	require.NoError(t, err)
}

func TestCheckPilotEligible_hwidCooldown(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	cooldown := 60 * 24 * time.Hour
	hwid := "abc123"

	inside := []AnchorRecord{{
		AnchorType:  AnchorHWID,
		AnchorValue: hwid,
		Status:      StatusExpired,
		IssuedAt:    now.Add(-30 * 24 * time.Hour),
	}}
	err := checkPilotEligible(now, cooldown, inside, CheckInput{HWID: hwid})
	require.ErrorIs(t, err, ErrTrialHWIDUsed)

	outside := []AnchorRecord{{
		AnchorType:  AnchorHWID,
		AnchorValue: hwid,
		Status:      StatusExpired,
		IssuedAt:    now.Add(-90 * 24 * time.Hour),
	}}
	err = checkPilotEligible(now, cooldown, outside, CheckInput{HWID: hwid})
	require.NoError(t, err)
}

func TestCheckPilotEligible_usdtWallet(t *testing.T) {
	now := time.Now().UTC()
	anchors := []AnchorRecord{{
		AnchorType:  AnchorUSDTTx,
		AnchorValue: "0xdead",
		Status:      StatusConverted,
		IssuedAt:    now,
	}}
	err := checkPilotEligible(now, 60*24*time.Hour, anchors, CheckInput{USDTTx: "0xdead"})
	require.ErrorIs(t, err, ErrTrialWalletUsed)
}

func TestRegistry_recordAndDenySecondPilot(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/trial.json"
	reg := New(path, 60*24*time.Hour)

	dep := "550e8400-e29b-41d4-a716-446655440000"
	validUntil := time.Now().UTC().Add(14 * 24 * time.Hour)

	require.NoError(t, reg.CheckPilotEligible(CheckInput{TelegramID: "999", DeploymentID: dep}))
	require.NoError(t, reg.RecordPilotIssue(RecordInput{
		TelegramID:   "999",
		DeploymentID: dep,
		LicenseKey:   dep,
		ValidUntil:   validUntil,
	}))

	err := reg.CheckPilotEligible(CheckInput{TelegramID: "999", DeploymentID: "660e8400-e29b-41d4-a716-446655440001"})
	require.ErrorIs(t, err, ErrTrialTelegramUsed)
}

func TestRegistry_recordHWIDBlocksEligible(t *testing.T) {
	dir := t.TempDir()
	reg := New(dir+"/trial.json", 60*24*time.Hour)

	dep := "550e8400-e29b-41d4-a716-446655440000"
	require.NoError(t, reg.RecordHWID(dep, "hwid-a"))

	err := reg.CheckPilotEligible(CheckInput{HWID: "hwid-a", DeploymentID: "660e8400-e29b-41d4-a716-446655440001"})
	require.ErrorIs(t, err, ErrTrialHWIDUsed)
}

func TestRegistry_forceOverrideAudit(t *testing.T) {
	dir := t.TempDir()
	reg := New(dir+"/trial.json", 60*24*time.Hour)

	dep := "550e8400-e29b-41d4-a716-446655440000"
	require.NoError(t, reg.RecordPilotIssue(RecordInput{
		TelegramID:   "1",
		DeploymentID: dep,
		LicenseKey:   dep,
		ValidUntil:   time.Now().UTC().Add(7 * 24 * time.Hour),
	}))
	require.NoError(t, reg.RecordPilotIssue(RecordInput{
		TelegramID:   "1",
		DeploymentID: "660e8400-e29b-41d4-a716-446655440001",
		LicenseKey:   "660e8400-e29b-41d4-a716-446655440001",
		ValidUntil:   time.Now().UTC().Add(7 * 24 * time.Hour),
		Force:        true,
		ForceReason:  "disk replaced",
		Operator:     "vendor",
	}))

	data, err := os.ReadFile(reg.Path())
	require.NoError(t, err)
	require.Contains(t, string(data), "disk replaced")
}

func TestRegistry_concurrentRecord(t *testing.T) {
	dir := t.TempDir()
	reg := New(dir+"/trial.json", 60*24*time.Hour)

	const workers = 8
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		i := i
		go func() {
			dep := "550e8400-e29b-41d4-a716-44665544000" + string(rune('0'+i))
			errCh <- reg.RecordPilotIssue(RecordInput{
				TelegramID:   "tg-" + string(rune('a'+i)),
				DeploymentID: dep,
				LicenseKey:   dep,
				ValidUntil:   time.Now().UTC().Add(24 * time.Hour),
			})
		}()
	}
	for i := 0; i < workers; i++ {
		require.NoError(t, <-errCh)
	}
}

func TestRegistry_markConverted(t *testing.T) {
	dir := t.TempDir()
	reg := New(dir+"/trial.json", 60*24*time.Hour)

	dep := "550e8400-e29b-41d4-a716-446655440000"
	require.NoError(t, reg.RecordPilotIssue(RecordInput{
		TelegramID:   "42",
		DeploymentID: dep,
		LicenseKey:   dep,
		ValidUntil:   time.Now().UTC().Add(24 * time.Hour),
	}))
	require.NoError(t, reg.MarkConverted(dep))

	err := reg.CheckPilotEligible(CheckInput{TelegramID: "42"})
	require.ErrorIs(t, err, ErrTrialTelegramUsed)
}

func TestValidateForceOverride(t *testing.T) {
	t.Setenv(EnvForceEnabled, "")
	err := ValidateForceOverride(true, "reason")
	require.ErrorIs(t, err, ErrForceNotAllowed)

	t.Setenv(EnvForceEnabled, "1")
	err = ValidateForceOverride(true, "")
	require.ErrorIs(t, err, ErrForceReason)

	err = ValidateForceOverride(true, "ok")
	require.NoError(t, err)

	require.NoError(t, ValidateForceOverride(false, ""))
}
