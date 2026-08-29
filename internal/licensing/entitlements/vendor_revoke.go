package entitlements

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ActivationLicenseKey(claims *LicenseClaims) string {
	if claims == nil {
		return ""
	}
	if key := strings.TrimSpace(claims.Subject); key != "" {
		return key
	}
	return strings.TrimSpace(claims.DeploymentID)
}

func VendorLicenseRevoked(ctx context.Context, pool *pgxpool.Pool, licenseKey string) (bool, error) {
	if pool == nil {
		return false, nil
	}
	licenseKey = strings.TrimSpace(licenseKey)
	if licenseKey == "" {
		return false, nil
	}
	var revoked bool
	err := pool.QueryRow(ctx, `
		SELECT revoked FROM vendor.licenses WHERE license_key = $1`, licenseKey).Scan(&revoked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return revoked, nil
}
