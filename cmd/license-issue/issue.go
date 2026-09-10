package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"ad-event-processor/internal/licenseissue"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/trialregistry"
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

func runIssue(opts *issueOptions, stderr io.Writer) (res issueResult, code int) {
	if opts.RecordHWID {
		return runRecordHWID(opts, stderr)
	}
	if opts.TrialMarkExpired {
		return runTrialMarkExpired(opts, stderr)
	}
	svc := licenseissue.New(licenseissue.Config{
		SKUFile:        opts.SKUFile,
		PrivateKeyFile: opts.PrivateKeyFile,
		KeyID:          opts.KID,
		TrialRegistry:  opts.TrialRegistry,
		MarkConverted:  opts.MarkConverted,
	})
	issued, err := svc.Issue(licenseissue.IssueRequest{
		SKUCode:          opts.SKUCode,
		Customer:         opts.Customer,
		DeploymentID:     opts.DeploymentID,
		Fingerprint:      opts.Fingerprint,
		HWIDV2:           opts.HWIDV2,
		TelegramID:       opts.TelegramID,
		USDTTx:           opts.USDTTx,
		ValidDays:        opts.ValidDays,
		Revoke:           opts.Revoke,
		Force:            opts.Force,
		ForceReason:      opts.ForceReason,
		Operator:         opts.Operator,
		ApprovePendingID: opts.ApprovePendingID,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "license-issue: %v\n", err)
		if errors.Is(err, trialregistry.ErrPendingNotFound) || errors.Is(err, trialregistry.ErrPendingNotOpen) {
			return issueResult{}, exitUsage
		}
		if errors.Is(err, trialregistry.ErrTrialTelegramUsed) ||
			errors.Is(err, trialregistry.ErrTrialHWIDUsed) ||
			errors.Is(err, trialregistry.ErrTrialWalletUsed) ||
			errors.Is(err, trialregistry.ErrOfferNotAccepted) ||
			errors.Is(err, trialregistry.ErrOfferVersionMismatch) {
			return issueResult{}, exitUsage
		}
		if strings.Contains(err.Error(), "required") {
			return issueResult{}, exitUsage
		}
		return issueResult{}, 1
	}

	if opts.MarkConverted && strings.TrimSpace(opts.Customer) == "" && issued.Token == "" {
		_, _ = fmt.Fprintf(stderr, "license-issue: marked converted deployment_id=%s\n", issued.DeploymentID)
		return issueResult{DeploymentID: issued.DeploymentID}, 0
	}

	if isPaidLicenseSKU(opts.SKUCode) && !opts.MarkConverted && issued.Token != "" {
		_, _ = fmt.Fprintf(stderr, "license-issue: warning: paid SKU %q issued without --mark-converted (deployment_id=%s)\n", opts.SKUCode, issued.DeploymentID)
	}

	return issueResult{
		Token:        issued.Token,
		DeploymentID: issued.DeploymentID,
		KeyID:        issued.KeyID,
		ValidUntil:   issued.ValidUntil,
		LicenseKey:   issued.LicenseKey,
	}, 0
}

func runTrialMarkExpired(opts *issueOptions, stderr io.Writer) (res issueResult, code int) {
	dep := strings.TrimSpace(opts.DeploymentID)
	if dep == "" {
		_, _ = fmt.Fprintln(stderr, "license-issue: --trial-mark-expired requires --deployment-id")
		return issueResult{}, exitUsage
	}
	reg := openRegistry(opts.TrialRegistry)
	if err := reg.MarkExpired(dep); err != nil {
		_, _ = fmt.Fprintf(stderr, "license-issue: mark expired: %v\n", err)
		return issueResult{}, 1
	}
	_, _ = fmt.Fprintf(stderr, "license-issue: marked expired deployment_id=%s\n", dep)
	return issueResult{DeploymentID: dep}, 0
}

func runRecordHWID(opts *issueOptions, stderr io.Writer) (res issueResult, code int) {
	dep := strings.TrimSpace(opts.DeploymentID)
	hwid := strings.TrimSpace(opts.HWIDV2)
	if dep == "" || hwid == "" {
		_, _ = fmt.Fprintln(stderr, "license-issue: --record-hwid requires --deployment-id and --hwid-v2")
		return issueResult{}, exitUsage
	}
	reg := openRegistry(opts.TrialRegistry)
	if err := reg.RecordHWID(dep, hwid); err != nil {
		_, _ = fmt.Fprintf(stderr, "license-issue: record hwid: %v\n", err)
		return issueResult{}, 1
	}
	_, _ = fmt.Fprintf(stderr, "license-issue: recorded hwid deployment_id=%s\n", dep)
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

func writeIssueOutput(res *issueResult, outFile string, stderr io.Writer) error {
	if strings.TrimSpace(outFile) != "" {
		if err := os.WriteFile(outFile, []byte(res.Token), 0o600); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stderr, "license-issue: wrote JWT to %s (kid=%s deployment_id=%s valid_until=%s)\n",
			outFile, res.KeyID, res.DeploymentID, res.ValidUntil.Format(time.RFC3339))
		return nil
	}
	if res.Token != "" {
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", res.Token)
	}
	if !res.ValidUntil.IsZero() {
		_, _ = fmt.Fprintf(stderr, "kid=%s deployment_id=%s valid_until=%s\n", res.KeyID, res.DeploymentID, res.ValidUntil.Format(time.RFC3339))
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
	fs.StringVar(&opts.TrialRegistry, "trial-registry", "", "trial anchor registry file (default VENDOR_TRIAL_REGISTRY or deploy/vendor/trial_registry.json)")
	fs.BoolVar(&opts.RecordHWID, "record-hwid", false, "record HWID anchor without issuing JWT")
	fs.BoolVar(&opts.TrialMarkExpired, "trial-mark-expired", false, "mark deployment pilot anchors expired in trial registry")
	fs.BoolVar(&opts.MarkConverted, "mark-converted", false, "mark deployment converted in trial registry")
	fs.BoolVar(&opts.Force, "force", false, "bypass pilot eligibility (requires VENDOR_TRIAL_FORCE=1)")
	fs.StringVar(&opts.ForceReason, "force-reason", "", "audit reason when --force is set")
	fs.StringVar(&opts.Operator, "operator", "", "vendor operator id for force audit")
	fs.StringVar(&opts.ApprovePendingID, "approve-pending", "", "approve pending trial request id and issue pilot JWT")

	if err := fs.Parse(args); err != nil {
		return issueOptions{}, err
	}
	return opts, nil
}
