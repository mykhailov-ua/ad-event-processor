package licenseissue

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/trialregistry"

	"github.com/stretchr/testify/require"
)

func TestIssue_pilot_withoutOfferAcceptance_holdout(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "trial.json")

	svc := New(Config{
		SKUFile:        filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"),
		PrivateKeyFile: writeTestPrivateKeyFile(t),
		TrialRegistry:  regPath,
	})

	_, err := svc.Issue(IssueRequest{
		SKUCode:    licensing.SKUCodePilot,
		Customer:   "Buyer",
		TelegramID: "7001",
	})
	require.ErrorIs(t, err, trialregistry.ErrOfferNotAccepted)
}

func TestIssue_pilot_forceWithoutOfferAcceptance_holdout(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "trial.json")
	t.Setenv(trialregistry.EnvForceEnabled, "1")

	svc := New(Config{
		SKUFile:        filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"),
		PrivateKeyFile: writeTestPrivateKeyFile(t),
		TrialRegistry:  regPath,
	})

	_, err := svc.Issue(IssueRequest{
		SKUCode:     licensing.SKUCodePilot,
		Customer:    "Buyer",
		TelegramID:  "7003",
		Force:       true,
		ForceReason: "support approved",
	})
	require.ErrorIs(t, err, trialregistry.ErrOfferNotAccepted)
}

func TestIssue_pilot_afterAcceptOffer(t *testing.T) {
	regPath := filepath.Join(t.TempDir(), "trial.json")
	reg := trialregistry.New(regPath, 0)
	require.NoError(t, reg.AcceptOffer("7002", trialregistry.CurrentOfferVersion(), trialregistry.AcceptSourceTelegram))

	svc := New(Config{
		SKUFile:        filepath.Join("..", "..", "deploy", "vendor", "sku.yaml"),
		PrivateKeyFile: writeTestPrivateKeyFile(t),
		TrialRegistry:  regPath,
	})

	res, err := svc.Issue(IssueRequest{
		SKUCode:    licensing.SKUCodePilot,
		Customer:   "Buyer",
		TelegramID: "7002",
	})
	require.NoError(t, err)
	require.NotEmpty(t, res.Token)
}

func writeTestPrivateKeyFile(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "license_private.key")
	seed := priv.Seed()
	encoded := hex.EncodeToString(seed)
	require.NoError(t, os.WriteFile(path, []byte(encoded+"\n"), 0o600))
	return path
}
