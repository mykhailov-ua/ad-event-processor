package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bidshard/ad-event-processor/internal/payment/db"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const metadataPurposeLicenseActivation = "license_activation"

func intentMetadataString(raw []byte, key string) string {
	if len(raw) == 0 || key == "" {
		return ""
	}
	var meta map[string]string
	if err := coldpath.UnmarshalJSON(raw, &meta); err != nil {
		return ""
	}
	return meta[key]
}

func maybeActivateLicenseFromIntent(ctx context.Context, tx pgx.Tx, intent db.PaymentPaymentIntent) error {
	if intentMetadataString(intent.Metadata, "purpose") != metadataPurposeLicenseActivation {
		return nil
	}
	deploymentID := intentMetadataString(intent.Metadata, "deployment_id")
	if deploymentID == "" {
		deploymentID = uuid.New().String()
	}
	licenseID := intentMetadataString(intent.Metadata, "license_id")
	if licenseID == "" {
		licenseID = uuid.New().String()
	}
	planCode := intentMetadataString(intent.Metadata, "plan_code")
	if planCode == "" {
		planCode = "crypto_topup"
	}
	validUntil := time.Now().UTC().Add(365 * 24 * time.Hour)
	entitlements := intentMetadataString(intent.Metadata, "entitlements_json")
	if entitlements == "" {
		entitlements = "{}"
	}
	depUUID, err := uuid.Parse(deploymentID)
	if err != nil {
		return fmt.Errorf("license activation deployment_id: %w", err)
	}
	licUUID, err := uuid.Parse(licenseID)
	if err != nil {
		return fmt.Errorf("license activation license_id: %w", err)
	}
	if !json.Valid([]byte(entitlements)) {
		entitlements = "{}"
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO billing.license_status (
			deployment_id, license_id, plan_code, valid_until, state, entitlements_json, last_verified_at, last_refresh_error
		) VALUES ($1, $2, $3, $4, 'ACTIVE', $5::jsonb, NOW(), '')
		ON CONFLICT (deployment_id) DO UPDATE SET
			license_id = EXCLUDED.license_id,
			plan_code = EXCLUDED.plan_code,
			valid_until = EXCLUDED.valid_until,
			state = 'ACTIVE',
			entitlements_json = EXCLUDED.entitlements_json,
			last_verified_at = NOW(),
			last_refresh_error = ''`,
		pgtype.UUID{Bytes: depUUID, Valid: true},
		pgtype.UUID{Bytes: licUUID, Valid: true},
		planCode,
		validUntil,
		entitlements,
	)
	return err
}
