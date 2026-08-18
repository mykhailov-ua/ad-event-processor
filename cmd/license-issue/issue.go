package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/internal/licensing"
	"github.com/bidshard/ad-event-processor/internal/trialregistry"

	"github.com/google/uuid"
)

const exitUsage = 2

type issueOptions struct {
	SKUFile          string
	SKUCode          string
	Customer         string
	DeploymentID     string
	Fingerprint      string
	HWIDV2           string
	KID              string
	Revoke           bool
	ValidDays        int
	PrivateKeyFile   string
	OutFile          string
	TelegramID       string
	USDTTx           string
	TrialRegistry    string
	RecordHWID       bool
	TrialMarkExpired bool
	MarkConverted    bool
	Force            bool
	ForceReason      string
	Operator         string
	ApprovePendingID string
}

type issueResult struct {
	Token        string
	DeploymentID string
	KeyID        string
	ValidUntil   time.Time
	LicenseKey   string
}

func runIssue(opts issueOptions, stderr io.Writer) (issueResult, int) {
	if opts.RecordHWID {
		return runRecordHWID(opts, stderr)
	}
	if opts.TrialMarkExpired {
		return runTrialMarkExpired(opts, stderr)
	}
	if pendingID := strings.TrimSpace(opts.ApprovePendingID); pendingID != "" {
		reg := openRegistry(opts.TrialRegistry)
		pending, err := reg.PreparePendingIssue(pendingID, opts.DeploymentID)
		if err != nil {
			fmt.Fprintf(stderr, "license-issue: approve pending: %v\n", err)
			if err == trialregistry.ErrPendingNotFound || err == trialregistry.ErrPendingNotOpen {
				return issueResult{}, exitUsage
			}
			return issueResult{}, 1
		}
		if strings.TrimSpace(opts.TelegramID) == "" {
			opts.TelegramID = pending.TelegramID
		}
		if strings.TrimSpace(opts.DeploymentID) == "" {
			opts.DeploymentID = pending.DeploymentID
		}
		if strings.TrimSpace(opts.Customer) == "" {
			if user := strings.TrimSpace(pending.TelegramUsername); user != "" {
				opts.Customer = "@" + strings.TrimPrefix(user, "@")
			} else {
				opts.Customer = "telegram:" + pending.TelegramID
			}
		}
		if !isPilotSKU(opts.SKUCode) {
			opts.SKUCode = licensing.SKUCodePilot
		}
	}

	if strings.TrimSpace(opts.Customer) == "" && !opts.MarkConverted {
		fmt.Fprintln(stderr, "license-issue: --customer is required")
		return issueResult{}, exitUsage
	}

	reg := openRegistry(opts.TrialRegistry)

	if opts.MarkConverted && strings.TrimSpace(opts.Customer) == "" {
		dep := strings.TrimSpace(opts.DeploymentID)
		if dep == "" {
			fmt.Fprintln(stderr, "license-issue: --deployment-id is required with --mark-converted")
			return issueResult{}, exitUsage
		}
		if err := reg.MarkConverted(dep); err != nil {
			fmt.Fprintf(stderr, "license-issue: mark converted: %v\n", err)
			return issueResult{}, 1
		}
		fmt.Fprintf(stderr, "license-issue: marked converted deployment_id=%s\n", dep)
		return issueResult{DeploymentID: dep}, 0
	}

	if err := trialregistry.ValidateForceOverride(opts.Force, opts.ForceReason); err != nil {
		fmt.Fprintf(stderr, "license-issue: %v\n", err)
		return issueResult{}, 1
	}

	keyID := strings.TrimSpace(opts.KID)
	if keyID == "" {
		keyID = licensing.DefaultLicenseKeyID
	}

	privPath := licensing.ResolvePrivateKeyFileForKID(keyID, strings.TrimSpace(opts.PrivateKeyFile))
	privBytes, err := os.ReadFile(privPath)
	if err != nil {
		fmt.Fprintf(stderr, "license-issue: read private key %s: %v\n", privPath, err)
		return issueResult{}, 1
	}
	priv, err := licensing.ParsePrivateKey(privBytes)
	if err != nil {
		fmt.Fprintf(stderr, "license-issue: parse private key: %v\n", err)
		return issueResult{}, 1
	}

	doc, err := licensing.LoadSKUFile(opts.SKUFile)
	if err != nil {
		fmt.Fprintf(stderr, "license-issue: load SKU file: %v\n", err)
		return issueResult{}, 1
	}
	sku, err := doc.GetSKU(opts.SKUCode)
	if err != nil {
		fmt.Fprintf(stderr, "license-issue: %v\n", err)
		return issueResult{}, 1
	}
	if opts.ValidDays > 0 {
		sku.ValidDays = opts.ValidDays
	}

	depID := strings.TrimSpace(opts.DeploymentID)
	if depID == "" {
		depID = uuid.NewString()
	}
	licenseID := uuid.NewString()

	hwid := strings.TrimSpace(opts.HWIDV2)
	if isPilotSKU(opts.SKUCode) {
		check := trialregistry.CheckInput{
			TelegramID:   opts.TelegramID,
			HWID:         hwid,
			USDTTx:       opts.USDTTx,
			DeploymentID: depID,
		}
		if !opts.Force {
			if err := reg.CheckPilotEligible(check); err != nil {
				fmt.Fprintf(stderr, "license-issue: pilot denied deployment_id=%s: %v\n", depID, err)
				return issueResult{}, exitUsage
			}
		}
	}

	claims := sku.BuildClaims(licensing.IssueLicenseInput{
		SKUCode:      sku.Code,
		CustomerName: opts.Customer,
		DeploymentID: depID,
		LicenseID:    licenseID,
		Fingerprint:  strings.TrimSpace(opts.Fingerprint),
		HWIDHash:     hwid,
		ValidFrom:    time.Now().UTC(),
	})
	if opts.Revoke {
		claims.Revoked = true
		claims.ValidUntil = time.Now().UTC().Add(-time.Hour)
		claims.ValidFrom = claims.ValidUntil.Add(-24 * time.Hour)
	}

	token, err := licensing.SignJWT(claims, priv, keyID)
	if err != nil {
		fmt.Fprintf(stderr, "license-issue: sign: %v\n", err)
		return issueResult{}, 1
	}

	if isPilotSKU(opts.SKUCode) {
		if err := reg.RecordPilotIssue(trialregistry.RecordInput{
			TelegramID:   opts.TelegramID,
			HWID:         hwid,
			USDTTx:       opts.USDTTx,
			DeploymentID: depID,
			LicenseKey:   licenseKeyFromClaims(claims),
			ValidUntil:   claims.ValidUntil,
			Force:        opts.Force,
			ForceReason:  opts.ForceReason,
			Operator:     opts.Operator,
		}); err != nil {
			fmt.Fprintf(stderr, "license-issue: record pilot issue: %v\n", err)
			return issueResult{}, 1
		}
	}

	if !isPilotSKU(opts.SKUCode) && opts.MarkConverted {
		if err := reg.MarkConverted(depID); err != nil {
			fmt.Fprintf(stderr, "license-issue: mark converted after issue: %v\n", err)
			return issueResult{}, 1
		}
		fmt.Fprintf(stderr, "license-issue: marked converted deployment_id=%s\n", depID)
	} else if isPaidLicenseSKU(opts.SKUCode) && !opts.MarkConverted {
		fmt.Fprintf(stderr, "license-issue: warning: paid SKU %q issued without --mark-converted (deployment_id=%s)\n", opts.SKUCode, depID)
	}

	return issueResult{
		Token:        token,
		DeploymentID: depID,
		KeyID:        keyID,
		ValidUntil:   claims.ValidUntil,
		LicenseKey:   licenseKeyFromClaims(claims),
	}, 0
}

func runTrialMarkExpired(opts issueOptions, stderr io.Writer) (issueResult, int) {
	dep := strings.TrimSpace(opts.DeploymentID)
	if dep == "" {
		fmt.Fprintln(stderr, "license-issue: --trial-mark-expired requires --deployment-id")
		return issueResult{}, exitUsage
	}
	reg := openRegistry(opts.TrialRegistry)
	if err := reg.MarkExpired(dep); err != nil {
		fmt.Fprintf(stderr, "license-issue: mark expired: %v\n", err)
		return issueResult{}, 1
	}
	fmt.Fprintf(stderr, "license-issue: marked expired deployment_id=%s\n", dep)
	return issueResult{DeploymentID: dep}, 0
}

func runRecordHWID(opts issueOptions, stderr io.Writer) (issueResult, int) {
	dep := strings.TrimSpace(opts.DeploymentID)
	hwid := strings.TrimSpace(opts.HWIDV2)
	if dep == "" || hwid == "" {
		fmt.Fprintln(stderr, "license-issue: --record-hwid requires --deployment-id and --hwid-v2")
		return issueResult{}, exitUsage
	}
	reg := openRegistry(opts.TrialRegistry)
	if err := reg.RecordHWID(dep, hwid); err != nil {
		fmt.Fprintf(stderr, "license-issue: record hwid: %v\n", err)
		return issueResult{}, 1
	}
	fmt.Fprintf(stderr, "license-issue: recorded hwid deployment_id=%s\n", dep)
	return issueResult{DeploymentID: dep}, 0
}

func openRegistry(pathOverride string) *trialregistry.Registry {
	cfg := trialregistry.ConfigFromEnv()
	if path := strings.TrimSpace(pathOverride); path != "" {
		cfg.RegistryPath = path
	}
	return trialregistry.NewFromConfig(cfg)
}

func isPilotSKU(code string) bool {
	return strings.EqualFold(strings.TrimSpace(code), licensing.SKUCodePilot)
}

func isPaidLicenseSKU(code string) bool {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "", licensing.SKUCodePilot, licensing.SKUCodeLicense:
		return false
	default:
		return true
	}
}

func licenseKeyFromClaims(claims licensing.LicenseClaims) string {
	if sub := strings.TrimSpace(claims.Subject); sub != "" {
		return sub
	}
	return strings.TrimSpace(claims.DeploymentID)
}

func writeIssueOutput(res issueResult, outFile string, stderr io.Writer) error {
	if strings.TrimSpace(outFile) != "" {
		if err := os.WriteFile(outFile, []byte(res.Token), 0o600); err != nil {
			return err
		}
		fmt.Fprintf(stderr, "license-issue: wrote JWT to %s (kid=%s deployment_id=%s valid_until=%s)\n",
			outFile, res.KeyID, res.DeploymentID, res.ValidUntil.Format(time.RFC3339))
		return nil
	}
	if res.Token != "" {
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", res.Token)
	}
	if !res.ValidUntil.IsZero() {
		fmt.Fprintf(stderr, "kid=%s deployment_id=%s valid_until=%s\n", res.KeyID, res.DeploymentID, res.ValidUntil.Format(time.RFC3339))
	}
	return nil
}

func parseFlags(args []string) (issueOptions, error) {
	fs := flag.NewFlagSet("license-issue", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	opts := issueOptions{}
	fs.StringVar(&opts.SKUFile, "sku-file", "deploy/vendor/sku.yaml", "path to SKU catalog")
	fs.StringVar(&opts.SKUCode, "sku", "pilot", "SKU code")
	fs.StringVar(&opts.Customer, "customer", "", "customer display name (required for JWT issue)")
	fs.StringVar(&opts.DeploymentID, "deployment-id", "", "deployment UUID (generated if empty)")
	fs.StringVar(&opts.Fingerprint, "fingerprint", "", "host fingerprint for hard bind (from customer support bundle)")
	fs.StringVar(&opts.HWIDV2, "hwid-v2", "", "host HWID v2 (Argon2id) for hard bind; preferred over --fingerprint")
	fs.StringVar(&opts.KID, "kid", licensing.DefaultLicenseKeyID, "JWT key id (kid); uses deploy/vendor/keys/<kid>/ when set")
	fs.BoolVar(&opts.Revoke, "revoke", false, "issue revocation JWT (valid_until in past, revoked=true)")
	fs.IntVar(&opts.ValidDays, "days", 0, "override valid_days from SKU")
	fs.StringVar(&opts.PrivateKeyFile, "private-key", "", "Ed25519 private key file (hex seed)")
	fs.StringVar(&opts.OutFile, "out", "", "write JWT to file instead of stdout")
	fs.StringVar(&opts.TelegramID, "telegram-id", "", "buyer Telegram user id (pilot trial registry)")
	fs.StringVar(&opts.USDTTx, "usdt-tx", "", "USDT wallet or tx id (pilot trial registry)")
	fs.StringVar(&opts.TrialRegistry, "trial-registry", "", "trial anchor registry file (default BIDSHARD_VENDOR_TRIAL_REGISTRY or deploy/vendor/trial_registry.json)")
	fs.BoolVar(&opts.RecordHWID, "record-hwid", false, "record HWID anchor without issuing JWT")
	fs.BoolVar(&opts.TrialMarkExpired, "trial-mark-expired", false, "mark deployment pilot anchors expired in trial registry")
	fs.BoolVar(&opts.MarkConverted, "mark-converted", false, "mark deployment converted in trial registry")
	fs.BoolVar(&opts.Force, "force", false, "bypass pilot eligibility (requires BIDSHARD_VENDOR_TRIAL_FORCE=1)")
	fs.StringVar(&opts.ForceReason, "force-reason", "", "audit reason when --force is set")
	fs.StringVar(&opts.Operator, "operator", "", "vendor operator id for force audit")
	fs.StringVar(&opts.ApprovePendingID, "approve-pending", "", "approve pending trial request id and issue pilot JWT")

	if err := fs.Parse(args); err != nil {
		return issueOptions{}, err
	}
	return opts, nil
}
