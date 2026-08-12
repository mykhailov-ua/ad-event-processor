package licensing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func activationLicenseKey(claims *LicenseClaims) string {
	if claims == nil {
		return ""
	}
	if key := stringsTrim(claims.Subject); key != "" {
		return key
	}
	return stringsTrim(claims.DeploymentID)
}

func stringsTrim(s string) string {
	return strings.TrimSpace(s)
}

// CheckHostActivation verifies bind mode (hard fingerprint or multi-host activation cap).
func CheckHostActivation(ctx context.Context, pool *pgxpool.Pool, claims *LicenseClaims, fingerprint string) error {
	if claims == nil {
		return ErrInvalidTokenFormat
	}
	claims.Features = SanitizeFeaturesForSKU(claims.SKU, claims.Features)
	if !BindModeMulti(claims.Bind.Mode) {
		return VerifyDeploymentBind(claims, fingerprint)
	}
	if pool == nil {
		return fmt.Errorf("activation check requires database pool")
	}
	if fingerprint == "" {
		return ErrFingerprintRequired
	}

	licenseKey := activationLicenseKey(claims)
	if licenseKey == "" {
		return ErrInvalidTokenFormat
	}

	if err := ensureVendorLicenseRow(ctx, pool, claims, licenseKey); err != nil {
		return err
	}

	acts, err := listActivationRecords(ctx, pool, licenseKey)
	if err != nil {
		return err
	}

	var dep *DeploymentRecord
	depID := stringsTrim(claims.DeploymentID)
	if depID != "" {
		if parsed, err := uuid.Parse(depID); err == nil {
			fp, err := loadDeploymentFingerprint(ctx, pool, parsed)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			if fp != "" {
				dep = &DeploymentRecord{
					DeploymentID: depID,
					LicenseKey:   licenseKey,
					Fingerprint:  fp,
				}
			}
		}
	}

	maxAct := NormalizeMaxActivationsLimit(claims.Limits)
	decision := EvaluateActivate(fingerprint, licenseKey, maxAct, acts, dep)
	if !decision.Allow {
		switch decision.DenyReason {
		case ErrFingerprintRequired.Error():
			return ErrFingerprintRequired
		case ErrFingerprintMismatch.Error():
			return ErrFingerprintMismatch
		case ErrActivationLimit.Error():
			return ErrActivationLimit
		default:
			return errors.New(decision.DenyReason)
		}
	}

	if decision.BindActivation {
		if err := recordActivation(ctx, pool, licenseKey, fingerprint, depID); err != nil {
			return err
		}
	}
	return nil
}

func ensureVendorLicenseRow(ctx context.Context, pool *pgxpool.Pool, claims *LicenseClaims, licenseKey string) error {
	limitsJSON, err := json.Marshal(claims.Limits)
	if err != nil {
		return err
	}
	featuresJSON, err := json.Marshal(SanitizeFeaturesForSKU(claims.SKU, claims.Features))
	if err != nil {
		return err
	}
	maxAct := NormalizeMaxActivationsLimit(claims.Limits)
	_, err = pool.Exec(ctx, `
		INSERT INTO vendor.licenses (
			license_key, customer_name, plan_code, valid_from, valid_until, grace_days,
			limits_json, features_json, support_tier, revoked, max_activations
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,false,$10)
		ON CONFLICT (license_key) DO UPDATE SET
			plan_code = EXCLUDED.plan_code,
			valid_until = EXCLUDED.valid_until,
			limits_json = EXCLUDED.limits_json,
			features_json = EXCLUDED.features_json,
			max_activations = EXCLUDED.max_activations,
			updated_at = NOW()`,
		licenseKey,
		claims.CustomerName,
		claims.Plan,
		claims.ValidFrom,
		claims.ValidUntil,
		claims.GraceDays,
		limitsJSON,
		featuresJSON,
		claims.SupportTier,
		maxAct,
	)
	return err
}

func listActivationRecords(ctx context.Context, pool *pgxpool.Pool, licenseKey string) ([]ActivationRecord, error) {
	rows, err := pool.Query(ctx, `
		SELECT license_key, fingerprint, deployment_id::text
		FROM vendor.license_activations
		WHERE license_key = $1
		ORDER BY first_seen_at`, licenseKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ActivationRecord
	for rows.Next() {
		var rec ActivationRecord
		if err := rows.Scan(&rec.LicenseKey, &rec.Fingerprint, &rec.DeploymentID); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func loadDeploymentFingerprint(ctx context.Context, pool *pgxpool.Pool, deploymentID uuid.UUID) (string, error) {
	var fp string
	err := pool.QueryRow(ctx, `
		SELECT fingerprint FROM vendor.deployments WHERE deployment_id = $1`, deploymentID).Scan(&fp)
	return fp, err
}

func recordActivation(ctx context.Context, pool *pgxpool.Pool, licenseKey, fingerprint, deploymentID string) error {
	depUUID, err := uuid.Parse(deploymentID)
	if err != nil {
		return fmt.Errorf("invalid deployment id: %w", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO vendor.license_activations (license_key, fingerprint, deployment_id, first_seen_at, last_seen_at)
		VALUES ($1,$2,$3,NOW(),NOW())
		ON CONFLICT (license_key, fingerprint) DO UPDATE SET
			deployment_id = EXCLUDED.deployment_id,
			last_seen_at = NOW()`,
		licenseKey, fingerprint, depUUID)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO vendor.deployments (deployment_id, license_key, fingerprint, activated_at, last_seen_at)
		VALUES ($1,$2,$3,NOW(),NOW())
		ON CONFLICT (deployment_id) DO UPDATE SET
			license_key = EXCLUDED.license_key,
			fingerprint = EXCLUDED.fingerprint,
			last_seen_at = NOW()`,
		depUUID, licenseKey, fingerprint)
	return err
}
