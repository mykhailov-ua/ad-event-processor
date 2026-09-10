package licenseissue

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/licensing/entitlements"
	"ad-event-processor/internal/trialregistry"

	"github.com/google/uuid"
)

type Config struct {
	SKUFile          string
	PrivateKeyFile   string
	KeyID            string
	TrialRegistry    string
	RecordHWID       bool
	TrialMarkExpired bool
	MarkConverted    bool
}

type IssueRequest struct {
	SKUCode          string
	Customer         string
	DeploymentID     string
	Fingerprint      string
	HWIDV2           string
	TelegramID       string
	USDTTx           string
	ValidDays        int
	Revoke           bool
	Force            bool
	ForceReason      string
	Operator         string
	ApprovePendingID string
}

type IssueResult struct {
	Token        string
	DeploymentID string
	KeyID        string
	ValidUntil   time.Time
	LicenseKey   string
}

type Service struct {
	cfg Config
	reg *trialregistry.Registry
}

func New(cfg Config) *Service {
	regPath := strings.TrimSpace(cfg.TrialRegistry)
	regCfg := trialregistry.ConfigFromEnv()
	if regPath != "" {
		regCfg.RegistryPath = regPath
	}
	return &Service{
		cfg: cfg,
		reg: trialregistry.NewFromConfig(regCfg),
	}
}

func (s *Service) Registry() *trialregistry.Registry {
	return s.reg
}

func (s *Service) Issue(req IssueRequest) (IssueResult, error) {
	if s.cfg.RecordHWID {
		return IssueResult{}, fmt.Errorf("record-hwid is CLI-only")
	}
	if s.cfg.TrialMarkExpired {
		return IssueResult{}, fmt.Errorf("trial-mark-expired is CLI-only")
	}

	customer := strings.TrimSpace(req.Customer)

	if s.cfg.MarkConverted && customer == "" {
		dep := strings.TrimSpace(req.DeploymentID)
		if dep == "" {
			return IssueResult{}, fmt.Errorf("deployment_id is required with mark-converted")
		}
		if err := s.reg.MarkConverted(dep); err != nil {
			return IssueResult{}, err
		}
		return IssueResult{DeploymentID: dep}, nil
	}

	skuCode := strings.TrimSpace(req.SKUCode)
	if skuCode == "" {
		skuCode = licensing.SKUCodePilot
	}

	if pendingID := strings.TrimSpace(req.ApprovePendingID); pendingID != "" {
		pending, err := s.reg.PreparePendingIssue(pendingID, req.DeploymentID)
		if err != nil {
			return IssueResult{}, err
		}
		if strings.TrimSpace(req.TelegramID) == "" {
			req.TelegramID = pending.TelegramID
		}
		if strings.TrimSpace(req.DeploymentID) == "" {
			req.DeploymentID = pending.DeploymentID
		}
		if customer == "" {
			if user := strings.TrimSpace(pending.TelegramUsername); user != "" {
				customer = "@" + strings.TrimPrefix(user, "@")
			} else {
				customer = "telegram:" + pending.TelegramID
			}
		}
		if !isPilotSKU(skuCode) {
			skuCode = licensing.SKUCodePilot
		}
	}

	if customer == "" && !s.cfg.MarkConverted {
		return IssueResult{}, fmt.Errorf("customer is required")
	}

	if err := trialregistry.ValidateForceOverride(req.Force, req.ForceReason); err != nil {
		return IssueResult{}, err
	}

	keyID := strings.TrimSpace(s.cfg.KeyID)
	if keyID == "" {
		keyID = licensing.DefaultLicenseKeyID
	}

	privPath := licensing.ResolvePrivateKeyFileForKID(keyID, strings.TrimSpace(s.cfg.PrivateKeyFile))
	privBytes, err := os.ReadFile(privPath)
	if err != nil {
		return IssueResult{}, fmt.Errorf("read private key %s: %w", privPath, err)
	}
	priv, err := licensing.ParsePrivateKey(privBytes)
	if err != nil {
		return IssueResult{}, fmt.Errorf("parse private key: %w", err)
	}

	doc, err := licensing.LoadSKUFile(s.cfg.SKUFile)
	if err != nil {
		return IssueResult{}, err
	}
	sku, err := doc.GetSKU(skuCode)
	if err != nil {
		return IssueResult{}, err
	}
	if req.ValidDays > 0 {
		sku.ValidDays = req.ValidDays
	}

	depID := strings.TrimSpace(req.DeploymentID)
	if depID == "" {
		depID = uuid.NewString()
	}
	licenseID := uuid.NewString()

	if err := s.requirePilotOfferAcceptance(req, skuCode); err != nil {
		return IssueResult{}, err
	}

	hwid := strings.TrimSpace(req.HWIDV2)
	if isPilotSKU(skuCode) {
		check := trialregistry.CheckInput{
			TelegramID:   req.TelegramID,
			HWID:         hwid,
			USDTTx:       req.USDTTx,
			DeploymentID: depID,
		}
		if !req.Force {
			if err := s.reg.CheckPilotEligible(check); err != nil {
				return IssueResult{}, err
			}
		}
	}

	claims := sku.BuildClaims(licensing.IssueLicenseInput{
		SKUCode:      sku.Code,
		CustomerName: customer,
		DeploymentID: depID,
		LicenseID:    licenseID,
		Fingerprint:  strings.TrimSpace(req.Fingerprint),
		HWIDHash:     hwid,
		ValidFrom:    time.Now().UTC(),
	})
	if req.Revoke {
		claims.Revoked = true
		claims.ValidUntil = time.Now().UTC().Add(-time.Hour)
		claims.ValidFrom = claims.ValidUntil.Add(-24 * time.Hour)
	}

	token, err := licensing.SignJWT(claims, priv, keyID)
	if err != nil {
		return IssueResult{}, err
	}

	if isPilotSKU(skuCode) {
		if err := s.reg.RecordPilotIssue(trialregistry.RecordInput{
			TelegramID:   req.TelegramID,
			HWID:         hwid,
			USDTTx:       req.USDTTx,
			DeploymentID: depID,
			LicenseKey:   licenseKeyFromClaims(&claims),
			ValidUntil:   claims.ValidUntil,
			Force:        req.Force,
			ForceReason:  req.ForceReason,
			Operator:     req.Operator,
		}); err != nil {
			return IssueResult{}, err
		}
	}

	if !isPilotSKU(skuCode) && s.cfg.MarkConverted {
		if err := s.reg.MarkConverted(depID); err != nil {
			return IssueResult{}, err
		}
	}

	return IssueResult{
		Token:        token,
		DeploymentID: depID,
		KeyID:        keyID,
		ValidUntil:   claims.ValidUntil,
		LicenseKey:   licenseKeyFromClaims(&claims),
	}, nil
}

func (s *Service) Catalog() ([]CatalogSKU, error) {
	doc, err := licensing.LoadSKUFile(s.cfg.SKUFile)
	if err != nil {
		return nil, err
	}
	out := make([]CatalogSKU, 0, len(doc.SKUs))
	for _, sku := range doc.SKUs {
		if sku.Code == entitlements.DefaultSKUCode {
			continue
		}
		out = append(out, catalogFromSKU(sku))
	}
	return out, nil
}

func MapIssueError(err error) (status int, code string) {
	if err == nil {
		return 200, ""
	}
	switch {
	case errors.Is(err, trialregistry.ErrTrialTelegramUsed):
		return 409, "TRIAL_TELEGRAM_USED"
	case errors.Is(err, trialregistry.ErrTrialHWIDUsed):
		return 409, "TRIAL_HWID_USED"
	case errors.Is(err, trialregistry.ErrTrialWalletUsed):
		return 409, "TRIAL_WALLET_USED"
	case errors.Is(err, trialregistry.ErrPendingNotFound):
		return 404, "PENDING_NOT_FOUND"
	case errors.Is(err, trialregistry.ErrPendingNotOpen):
		return 409, "PENDING_NOT_OPEN"
	case errors.Is(err, trialregistry.ErrForceNotAllowed), errors.Is(err, trialregistry.ErrForceReason):
		return 400, "FORCE_INVALID"
	case errors.Is(err, trialregistry.ErrOfferNotAccepted):
		return 400, "OFFER_NOT_ACCEPTED"
	case errors.Is(err, trialregistry.ErrOfferVersionMismatch):
		return 400, "OFFER_VERSION_MISMATCH"
	default:
		if strings.Contains(err.Error(), "not found") {
			return 404, "NOT_FOUND"
		}
		if strings.Contains(err.Error(), "required") {
			return 400, "VALIDATION_ERROR"
		}
		return 500, "INTERNAL_ERROR"
	}
}

func isPilotSKU(code string) bool {
	return strings.EqualFold(strings.TrimSpace(code), licensing.SKUCodePilot)
}

func (s *Service) requirePilotOfferAcceptance(req IssueRequest, skuCode string) error {
	if !isPilotSKU(skuCode) {
		return nil
	}
	telegramID := strings.TrimSpace(req.TelegramID)
	if telegramID == "" {
		return fmt.Errorf("telegram_id is required for pilot")
	}
	accepted, err := s.reg.HasOfferAcceptance(telegramID, trialregistry.CurrentOfferVersion())
	if err != nil {
		return err
	}
	if !accepted {
		return trialregistry.ErrOfferNotAccepted
	}
	return nil
}

func licenseKeyFromClaims(claims *entitlements.LicenseClaims) string {
	if sub := strings.TrimSpace(claims.Subject); sub != "" {
		return sub
	}
	return strings.TrimSpace(claims.DeploymentID)
}
