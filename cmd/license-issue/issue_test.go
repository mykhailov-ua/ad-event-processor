package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/licensing"
	"github.com/bidshard/ad-event-processor/internal/trialregistry"

	"github.com/stretchr/testify/require"
)

func TestRunIssue_approvePending(t *testing.T) {
	privPath := writeTestPrivateKey(t)
	regPath := filepath.Join(t.TempDir(), "trial.json")
	t.Setenv(trialregistry.EnvRegistryPath, regPath)

	reg := trialregistry.New(regPath, 0)
	pending, err := reg.EnqueuePending(trialregistry.EnqueuePendingInput{
		TelegramID:       "9009",
		TelegramUsername: "buyer9009",
	})
	require.NoError(t, err)

	opts := issueOptions{
		SKUFile:          filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"),
		SKUCode:          licensing.SKUCodePilot,
		ApprovePendingID: pending.ID,
		PrivateKeyFile:   privPath,
	}
	res, code := runIssue(opts, &bytes.Buffer{})
	require.Equal(t, 0, code)
	require.NotEmpty(t, res.Token)
	require.NotEmpty(t, res.DeploymentID)

	_, code = runIssue(opts, &bytes.Buffer{})
	require.Equal(t, exitUsage, code)
}

func TestRunIssue_pilotDenyRepeatTelegram(t *testing.T) {
	privPath := writeTestPrivateKey(t)
	regPath := filepath.Join(t.TempDir(), "trial.json")
	t.Setenv(trialregistry.EnvRegistryPath, regPath)

	opts := issueOptions{
		SKUFile:        filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"),
		SKUCode:        licensing.SKUCodePilot,
		Customer:       "Buyer A",
		TelegramID:     "1001",
		PrivateKeyFile: privPath,
	}
	_, code := runIssue(opts, &bytes.Buffer{})
	require.Equal(t, 0, code)

	opts.Customer = "Buyer B"
	_, code = runIssue(opts, &bytes.Buffer{})
	require.Equal(t, exitUsage, code)
}

func TestRunIssue_pilotForceOverride(t *testing.T) {
	privPath := writeTestPrivateKey(t)
	regPath := filepath.Join(t.TempDir(), "trial.json")
	t.Setenv(trialregistry.EnvRegistryPath, regPath)
	t.Setenv(trialregistry.EnvForceEnabled, "1")

	opts := issueOptions{
		SKUFile:        filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"),
		SKUCode:        licensing.SKUCodePilot,
		Customer:       "Buyer A",
		TelegramID:     "2002",
		PrivateKeyFile: privPath,
	}
	_, code := runIssue(opts, &bytes.Buffer{})
	require.Equal(t, 0, code)

	opts.Customer = "Buyer B"
	opts.Force = true
	opts.ForceReason = "support approved"
	_, code = runIssue(opts, &bytes.Buffer{})
	require.Equal(t, 0, code)

	data, err := os.ReadFile(regPath)
	require.NoError(t, err)
	require.Contains(t, string(data), "support approved")
}

func TestRunIssue_forceWithoutEnvFails(t *testing.T) {
	t.Setenv(trialregistry.EnvForceEnabled, "")
	err := trialregistry.ValidateForceOverride(true, "reason")
	require.ErrorIs(t, err, trialregistry.ErrForceNotAllowed)
}

func TestRunIssue_recordHWID(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "trial.json")
	t.Setenv(trialregistry.EnvRegistryPath, regPath)

	dep := "550e8400-e29b-41d4-a716-446655440000"
	opts := issueOptions{
		RecordHWID:   true,
		DeploymentID: dep,
		HWIDV2:       "hwid-test-hash",
	}
	_, code := runIssue(opts, &bytes.Buffer{})
	require.Equal(t, 0, code)

	reg := trialregistry.New(regPath, 0)
	err := reg.CheckPilotEligible(trialregistry.CheckInput{
		HWID:         "hwid-test-hash",
		DeploymentID: "other",
	})
	require.ErrorIs(t, err, trialregistry.ErrTrialHWIDUsed)
}

func TestRunIssue_starterMarkConvertedDeniesPilot(t *testing.T) {
	privPath := writeTestPrivateKey(t)
	regPath := filepath.Join(t.TempDir(), "trial.json")
	t.Setenv(trialregistry.EnvRegistryPath, regPath)

	dep := "550e8400-e29b-41d4-a716-446655440000"
	pilot := issueOptions{
		SKUFile:        filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"),
		SKUCode:        licensing.SKUCodePilot,
		Customer:       "Trial Buyer",
		TelegramID:     "4004",
		DeploymentID:   dep,
		PrivateKeyFile: privPath,
	}
	_, code := runIssue(pilot, &bytes.Buffer{})
	require.Equal(t, 0, code)

	starter := issueOptions{
		SKUFile:        filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"),
		SKUCode:        licensing.SKUCodeStarter,
		Customer:       "Paid Buyer",
		DeploymentID:   dep,
		MarkConverted:  true,
		PrivateKeyFile: privPath,
	}
	_, code = runIssue(starter, &bytes.Buffer{})
	require.Equal(t, 0, code)

	pilot.Customer = "Another Trial"
	_, code = runIssue(pilot, &bytes.Buffer{})
	require.Equal(t, exitUsage, code)
}

func TestRunIssue_trialMarkExpired(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "trial.json")
	t.Setenv(trialregistry.EnvRegistryPath, regPath)

	dep := "550e8400-e29b-41d4-a716-446655440000"
	reg := trialregistry.New(regPath, 0)
	require.NoError(t, reg.RecordPilotIssue(trialregistry.RecordInput{
		TelegramID:   "5005",
		DeploymentID: dep,
		LicenseKey:   dep,
		ValidUntil:   time.Now().UTC().Add(24 * time.Hour),
	}))

	opts := issueOptions{
		TrialMarkExpired: true,
		DeploymentID:     dep,
	}
	_, code := runIssue(opts, &bytes.Buffer{})
	require.Equal(t, 0, code)

	err := reg.CheckPilotEligible(trialregistry.CheckInput{TelegramID: "5005"})
	require.ErrorIs(t, err, trialregistry.ErrTrialTelegramUsed)
}

func TestRunIssue_paidWithoutMarkConvertedWarns(t *testing.T) {
	privPath := writeTestPrivateKey(t)
	var stderr bytes.Buffer

	opts := issueOptions{
		SKUFile:        filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"),
		SKUCode:        licensing.SKUCodeStarter,
		Customer:       "Paid Buyer",
		DeploymentID:   "550e8400-e29b-41d4-a716-446655440001",
		PrivateKeyFile: privPath,
	}
	_, code := runIssue(opts, &stderr)
	require.Equal(t, 0, code)
	require.Contains(t, stderr.String(), "warning: paid SKU")
	require.Contains(t, stderr.String(), "without --mark-converted")
}

func TestRunIssue_starterSkipsPilotEligible(t *testing.T) {
	privPath := writeTestPrivateKey(t)
	regPath := filepath.Join(t.TempDir(), "trial.json")
	t.Setenv(trialregistry.EnvRegistryPath, regPath)

	opts := issueOptions{
		SKUFile:        filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"),
		SKUCode:        licensing.SKUCodeStarter,
		Customer:       "Paid Buyer",
		TelegramID:     "3003",
		PrivateKeyFile: privPath,
	}
	_, code := runIssue(opts, &bytes.Buffer{})
	require.Equal(t, 0, code)

	opts.Customer = "Paid Buyer 2"
	_, code = runIssue(opts, &bytes.Buffer{})
	require.Equal(t, 0, code)
}

func writeTestPrivateKey(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "license_private.key")
	seed := priv.Seed()
	encoded := hex.EncodeToString(seed)
	require.NoError(t, os.WriteFile(path, []byte(encoded+"\n"), 0o600))
	return path
}
