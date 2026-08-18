package licensing

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ActivationLicenseKey returns the customer-plane license anchor used in vendor.* tables.
func ActivationLicenseKey(claims *LicenseClaims) string {
	return activationLicenseKey(claims)
}

// VendorLicenseRevoked reports whether vendor.licenses.revoked is set for license_key.
func VendorLicenseRevoked(ctx context.Context, pool *pgxpool.Pool, licenseKey string) (bool, error) {
	if pool == nil {
		return false, nil
	}
	licenseKey = stringsTrim(licenseKey)
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
