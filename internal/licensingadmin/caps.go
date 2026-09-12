package licensingadmin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ad-event-processor/internal/licensing"
	entitlements "ad-event-processor/internal/licensing/entitlements"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CapHost interface {
	Pool() *pgxpool.Pool
	DeploymentLimits() (licensing.Limits, licensing.LicenseState, bool)
	ErrValidation(msg string) error
}

type LimitUsage struct {
	Used  uint64 `json:"used"`
	Limit uint64 `json:"limit"`
}

type DeploymentLimitUsage struct {
	Tenants     *LimitUsage `json:"tenants,omitempty"`
	APIKeys     *LimitUsage `json:"api_keys,omitempty"`
	Regions     *LimitUsage `json:"regions,omitempty"`
	EventsMonth *LimitUsage `json:"events_month,omitempty"`
}

func limitUnlimited(max uint64) bool {
	return max == 0 || max >= 999999
}

func requireActiveLicense(host CapHost, state licensing.LicenseState, ok bool) error {
	if !ok {
		return nil
	}
	if state == licensing.StateExpired || state == licensing.StateRevoked {
		return host.ErrValidation("license not active")
	}
	return nil
}

func recordCapReject(limit string) {
	if metrics.LicenseCapRejectedTotal != nil {
		metrics.LicenseCapRejectedTotal.WithLabelValues(limit).Inc()
	}
}

func EnforceDeploymentTenantCap(ctx context.Context, host CapHost) error {
	if host == nil || host.Pool() == nil {
		return nil
	}
	limits, state, ok := host.DeploymentLimits()
	if err := requireActiveLicense(host, state, ok); err != nil {
		return err
	}
	if !ok || limitUnlimited(limits.MaxTenants) {
		return nil
	}
	used, err := CountDeploymentTenants(ctx, host.Pool())
	if err != nil {
		return err
	}
	if used >= limits.MaxTenants {
		recordCapReject("tenants")
		return ErrDeploymentTenantLimit
	}
	return nil
}

func EnforceDeploymentAPIKeyCap(ctx context.Context, host CapHost) error {
	if host == nil || host.Pool() == nil {
		return nil
	}
	limits, state, ok := host.DeploymentLimits()
	if err := requireActiveLicense(host, state, ok); err != nil {
		return err
	}
	if !ok || limitUnlimited(limits.MaxAPIKeys) {
		return nil
	}
	used, err := CountDeploymentAPIKeys(ctx, host.Pool())
	if err != nil {
		return err
	}
	if used >= limits.MaxAPIKeys {
		recordCapReject("api_keys")
		return ErrDeploymentAPIKeyLimit
	}
	return nil
}

func EnforceDeploymentExportAllowed(host CapHost) error {
	if host == nil {
		return nil
	}
	limits, state, ok := host.DeploymentLimits()
	if err := requireActiveLicense(host, state, ok); err != nil {
		return err
	}
	if !ok || limits.MaxExportChunkBytes > 0 {
		return nil
	}
	recordCapReject("exports")
	return ErrDeploymentExportDisabled
}

func EnforceDeploymentRegionCap(ctx context.Context, host CapHost, regionCode int16) error {
	if host == nil || host.Pool() == nil {
		return nil
	}
	limits, state, ok := host.DeploymentLimits()
	if err := requireActiveLicense(host, state, ok); err != nil {
		return err
	}
	if !ok || limitUnlimited(limits.MaxRegions) {
		return nil
	}
	used, known, err := CountDeploymentRegions(ctx, host.Pool(), regionCode)
	if err != nil {
		return err
	}
	if known || used < limits.MaxRegions {
		return nil
	}
	recordCapReject("regions")
	return ErrDeploymentRegionLimit
}

func EnforceCostSyncNetworkCap(ctx context.Context, host CapHost, customerID uuid.UUID, addingNewNetwork bool) error {
	if host == nil {
		return nil
	}
	limits, state, ok := host.DeploymentLimits()
	if err := requireActiveLicense(host, state, ok); err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if entitlements.CostSyncNetworksDisabled(limits) {
		recordCapReject("cost_sync_networks")
		return ErrDeploymentCostSyncNetworkLimit
	}
	if entitlements.CostSyncNetworksUnlimited(limits) || !addingNewNetwork {
		return nil
	}
	if host.Pool() == nil || customerID == uuid.Nil {
		return nil
	}
	cap := entitlements.CostSyncNetworksCap(limits)
	if cap == 0 {
		return nil
	}
	used, err := CountCustomerCostSyncNetworks(ctx, host.Pool(), customerID)
	if err != nil {
		return err
	}
	if used >= cap {
		recordCapReject("cost_sync_networks")
		return ErrDeploymentCostSyncNetworkLimit
	}
	return nil
}

func EnforceDeploymentMonthlyEventsCap(ctx context.Context, host CapHost) error {
	if host == nil || host.Pool() == nil {
		return nil
	}
	limits, state, ok := host.DeploymentLimits()
	if err := requireActiveLicense(host, state, ok); err != nil {
		return err
	}
	if !ok || limitUnlimited(limits.MaxEventsPerMonth) {
		return nil
	}
	used, err := SumDeploymentAcceptedEventsMonth(ctx, host.Pool(), limits.QuotaResetTimezone)
	if err != nil {
		return err
	}
	if used >= limits.MaxEventsPerMonth {
		recordCapReject("events_month")
		return ErrDeploymentMonthlyEventsLimit
	}
	return nil
}

func CountDeploymentTenants(ctx context.Context, pool *pgxpool.Pool) (uint64, error) {
	var count int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM customers`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count customers: %w", err)
	}
	if count < 0 {
		return 0, nil
	}
	return uint64(count), nil
}

func CountCustomerCostSyncNetworks(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) (uint64, error) {
	if pool == nil || customerID == uuid.Nil {
		return 0, nil
	}
	var count int64
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM cost_sync_credentials WHERE customer_id = $1`,
		customerID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count cost sync credentials: %w", err)
	}
	if count < 0 {
		return 0, nil
	}
	return uint64(count), nil
}

func CountDeploymentAPIKeys(ctx context.Context, pool *pgxpool.Pool) (uint64, error) {
	var count int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM api_keys`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count api keys: %w", err)
	}
	if count < 0 {
		return 0, nil
	}
	return uint64(count), nil
}

func CountDeploymentRegions(ctx context.Context, pool *pgxpool.Pool, regionCode int16) (used uint64, regionKnown bool, err error) {
	var known bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM node_metric_buckets WHERE region_code = $1 LIMIT 1
		)`, regionCode).Scan(&known); err != nil {
		return 0, false, fmt.Errorf("region known check: %w", err)
	}
	var count int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(DISTINCT region_code) FROM node_metric_buckets`).Scan(&count); err != nil {
		if isUndefinedRelationErr(err) {
			if regionCode == 0 {
				return 0, true, nil
			}
			return 1, known, nil
		}
		return 0, false, fmt.Errorf("count regions: %w", err)
	}
	if count < 0 {
		count = 0
	}
	return uint64(count), known, nil
}

func SumDeploymentAcceptedEventsMonth(ctx context.Context, pool *pgxpool.Pool, timezone string) (uint64, error) {
	loc := time.UTC
	if timezone != "" {
		if parsed, err := time.LoadLocation(timezone); err == nil {
			loc = parsed
		}
	}
	now := time.Now().In(loc)
	period := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	pgPeriod := pgtype.Date{Time: period, Valid: true}

	var total pgtype.Int8
	err := pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(value), 0)::bigint
		FROM billing.usage_meters
		WHERE meter = $1 AND period = $2`, "accepted_events", pgPeriod).Scan(&total)
	if err != nil {
		if isUndefinedRelationErr(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("sum deployment accepted events month: %w", err)
	}
	if !total.Valid || total.Int64 < 0 {
		return 0, nil
	}
	return uint64(total.Int64), nil
}

func DeploymentLimitUsageSnapshot(ctx context.Context, host CapHost) (DeploymentLimitUsage, error) {
	var out DeploymentLimitUsage
	if host == nil || host.Pool() == nil {
		return out, nil
	}
	limits, _, ok := host.DeploymentLimits()
	if !ok {
		return out, nil
	}
	if !limitUnlimited(limits.MaxTenants) {
		used, err := CountDeploymentTenants(ctx, host.Pool())
		if err != nil {
			return out, err
		}
		out.Tenants = &LimitUsage{Used: used, Limit: limits.MaxTenants}
	}
	if !limitUnlimited(limits.MaxAPIKeys) {
		used, err := CountDeploymentAPIKeys(ctx, host.Pool())
		if err != nil {
			return out, err
		}
		out.APIKeys = &LimitUsage{Used: used, Limit: limits.MaxAPIKeys}
	}
	if !limitUnlimited(limits.MaxRegions) {
		used, _, err := CountDeploymentRegions(ctx, host.Pool(), 0)
		if err != nil {
			return out, err
		}
		out.Regions = &LimitUsage{Used: used, Limit: limits.MaxRegions}
	}
	if !limitUnlimited(limits.MaxEventsPerMonth) {
		used, err := SumDeploymentAcceptedEventsMonth(ctx, host.Pool(), limits.QuotaResetTimezone)
		if err != nil {
			return out, err
		}
		out.EventsMonth = &LimitUsage{Used: used, Limit: limits.MaxEventsPerMonth}
	}
	return out, nil
}

func isUndefinedRelationErr(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42P01"
}
