package campaign

import (
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/money"

	"github.com/jackc/pgx/v5/pgtype"
)

func parsePatchAttestationTTLSec(raw *int32) (int32, bool, error) {
	if raw == nil {
		return 0, false, nil
	}
	v := *raw
	if v < 60 || v > 900 {
		return 0, false, errValidation("attestation_ttl_sec must be between 60 and 900")
	}
	return v, true, nil
}

func parsePatchAttestationMode(raw *string) (domain.AttestationMode, bool, error) {
	if raw == nil {
		return domain.AttestationModeOff, false, nil
	}
	mode := domain.ParseAttestationMode(*raw)
	switch mode {
	case domain.AttestationModeOff, domain.AttestationModeLight, domain.AttestationModeStrict:
		return mode, true, nil
	default:
		return domain.AttestationModeOff, false, errValidation("attestation_mode must be off, light, or strict")
	}
}

func parsePatchConnTypePolicy(raw *string) (string, bool, error) {
	if raw == nil {
		return "", false, nil
	}
	s := strings.TrimSpace(*raw)
	switch s {
	case string(domain.ConnTypeBlockVPNHosting), string(domain.ConnTypeMobileOnly), string(domain.ConnTypeResidentialOnly):
		return s, true, nil
	default:
		return "", false, errValidation(fmt.Sprintf("invalid conn_type_policy %q", s))
	}
}

func parsePatchLinkSigningTTLSec(raw *int32) (int32, bool, error) {
	if raw == nil {
		return 0, false, nil
	}
	v := *raw
	if v < 60 || v > 3600 {
		return 0, false, errValidation("link_signing_ttl_sec must be between 60 and 3600")
	}
	return v, true, nil
}

func resolvePatchBudgetLimitMicro(req PatchCampaignRequest) (*int64, error) {
	if req.BudgetLimitMicro != nil {
		if *req.BudgetLimitMicro <= 0 {
			return nil, errValidation("budget must be positive")
		}
		v := *req.BudgetLimitMicro
		return &v, nil
	}
	if req.BudgetLimit != nil {
		v, err := money.ParseDecimal(strings.TrimSpace(*req.BudgetLimit))
		if err != nil || v <= 0 {
			return nil, errValidation("budget must be positive")
		}
		return &v, nil
	}
	return nil, nil
}

func parsePatchStatus(raw *string) (db.CampaignStatusType, bool, error) {
	if raw == nil {
		return "", false, nil
	}
	switch strings.ToUpper(strings.TrimSpace(*raw)) {
	case string(db.CampaignStatusTypeACTIVE):
		return db.CampaignStatusTypeACTIVE, true, nil
	case string(db.CampaignStatusTypePAUSED):
		return db.CampaignStatusTypePAUSED, true, nil
	default:
		return "", false, errValidation(fmt.Sprintf("invalid status %q", *raw))
	}
}

func timestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time.UTC()
	return &v
}

func toTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}
}

func ParsePatchAttestationTTLSec(raw *int32) (int32, bool, error) {
	return parsePatchAttestationTTLSec(raw)
}

func ParsePatchAttestationMode(raw *string) (domain.AttestationMode, bool, error) {
	return parsePatchAttestationMode(raw)
}

func ParsePatchConnTypePolicy(raw *string) (string, bool, error) {
	return parsePatchConnTypePolicy(raw)
}

func ParsePatchLinkSigningTTLSec(raw *int32) (int32, bool, error) {
	return parsePatchLinkSigningTTLSec(raw)
}

func ResolvePatchBudgetLimitMicro(req PatchCampaignRequest) (*int64, error) {
	return resolvePatchBudgetLimitMicro(req)
}

func ParsePatchStatus(raw *string) (db.CampaignStatusType, bool, error) {
	return parsePatchStatus(raw)
}

func TimestamptzPtr(t pgtype.Timestamptz) *time.Time {
	return timestamptzPtr(t)
}
