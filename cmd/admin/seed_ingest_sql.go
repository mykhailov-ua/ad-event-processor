package main

import (
	"fmt"
	"io"
	"strings"
)

func sqlLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func writeSeedIngestSQL(w io.Writer, count int) {
	if count < 1 {
		count = 100
	}

	fmt.Fprintln(w, "INSERT INTO customers (id, name, balance, currency, allowed_overdraft)")
	fmt.Fprintln(w, "VALUES")
	for i := 1; i <= count; i++ {
		sep := ","
		if i == count {
			sep = ""
		}
		fmt.Fprintf(
			w,
			"  ('%s', %s, %d, 'USD', 0)%s\n",
			seedCustomerUUID(i),
			sqlLiteral(seedCustomerName(i)),
			seedCustomerBalanceMicro(i),
			sep,
		)
	}
	fmt.Fprintln(w, "ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, balance = EXCLUDED.balance;")
	fmt.Fprintln(w)

	fmt.Fprintln(w, "INSERT INTO advertiser_brands (id, customer_id, name)")
	fmt.Fprintln(w, "VALUES")
	for i := 1; i <= count; i++ {
		sep := ","
		if i == count {
			sep = ""
		}
		fmt.Fprintf(
			w,
			"  ('%s', '%s', %s)%s\n",
			seedBrandUUID(i),
			seedCustomerUUID(i),
			sqlLiteral(seedBrandDisplayName(i)),
			sep,
		)
	}
	fmt.Fprintln(w, "ON CONFLICT (id) DO NOTHING;")
	fmt.Fprintln(w)

	fmt.Fprintln(w, "INSERT INTO brand_creatives (id, brand_id, name, landing_url, weight, status)")
	fmt.Fprintln(w, "VALUES")
	for i := 1; i <= count; i++ {
		sep := ","
		if i == count {
			sep = ""
		}
		fmt.Fprintf(
			w,
			"  ('%s', '%s', %s, 'https://trk.horizon-media.io/landing?cid={click_id}', %d, 'ACTIVE')%s\n",
			seedCreativeUUID(i),
			seedBrandUUID(i),
			sqlLiteral(seedCreativeDisplayName(i)),
			97+(i%13),
			sep,
		)
	}
	fmt.Fprintln(w, `ON CONFLICT (brand_id, name) DO UPDATE SET
  landing_url = EXCLUDED.landing_url,
  status = 'ACTIVE',
  updated_at = NOW();`)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "INSERT INTO campaigns (id, name, budget_limit, status, customer_id, pacing_mode, daily_budget, timezone, freq_limit, freq_window, brand_id, target_url)")
	fmt.Fprintln(w, "VALUES")
	for i := 1; i <= count; i++ {
		sep := ","
		if i == count {
			sep = ""
		}
		budgetLimit := int64(4_200_000_000 + (int64(i%17) * 650_000_000) + (int64(i%9) * 384_729))
		dailyBudget := int64(380_000_000 + (int64(i%11) * 95_000_000) + (int64(i%6) * 18_473))
		fmt.Fprintf(
			w,
			"  ('%s', %s, %d, 'ACTIVE', '%s', 'ASAP', %d, 'UTC', 100000000, 3600, '%s', 'https://trk.horizon-media.io/landing?cid={click_id}')%s\n",
			seedCampaignUUID(i),
			sqlLiteral(seedCampaignName(i)),
			budgetLimit,
			seedCustomerUUID(i),
			dailyBudget,
			seedBrandUUID(i),
			sep,
		)
	}
	fmt.Fprintln(w, `ON CONFLICT (id) DO UPDATE SET
  current_spend = 0,
  status = 'ACTIVE',
  budget_limit = EXCLUDED.budget_limit,
  brand_id = EXCLUDED.brand_id,
  target_url = EXCLUDED.target_url;`)
	fmt.Fprintln(w)

	fmt.Fprintf(w, `INSERT INTO billing.license_status (
    deployment_id, license_id, plan_code, valid_until, state, entitlements_json, last_verified_at
) VALUES (
    '%s',
    '%s',
    'pilot',
    NOW() + INTERVAL '365 days',
    'ACTIVE',
    '{"limits":{"max_active_campaigns":1000,"max_rps":200000,"max_requests_per_day":0,"max_events_per_month":0,"max_regions":4,"max_api_keys":10,"max_export_chunk_bytes":10485760,"quota_reset_timezone":"UTC"},"features":{"rtb_live":true,"ml_fraud_boost":true,"multi_region":true,"slot_migration":true}}'::jsonb,
    NOW()
)
ON CONFLICT (deployment_id) DO UPDATE SET
    state = EXCLUDED.state,
    valid_until = EXCLUDED.valid_until,
    entitlements_json = EXCLUDED.entitlements_json,
    last_verified_at = EXCLUDED.last_verified_at;
`, seedDeploymentUUID(), seedLicenseRecordUUID())
}

func writeSeedPrepTestSQL(w io.Writer, count int) {
	if count < 1 {
		count = 100
	}

	fmt.Fprintln(w, "INSERT INTO customers (id, name, balance, currency, allowed_overdraft)")
	fmt.Fprintln(w, "VALUES")
	for i := 1; i <= count; i++ {
		sep := ","
		if i == count {
			sep = ""
		}
		fmt.Fprintf(
			w,
			"  ('%s', %s, %d, 'USD', 0)%s\n",
			seedCustomerUUID(i),
			sqlLiteral(seedCustomerName(i)),
			seedCustomerBalanceMicro(i),
			sep,
		)
	}
	fmt.Fprintln(w, `ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  balance = EXCLUDED.balance;`)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "INSERT INTO campaigns (id, name, budget_limit, status, customer_id, pacing_mode, daily_budget, timezone, freq_limit, freq_window)")
	fmt.Fprintln(w, "VALUES")
	for i := 1; i <= count; i++ {
		sep := ","
		if i == count {
			sep = ""
		}
		budgetLimit := int64(4_200_000_000 + (int64(i%17) * 650_000_000) + (int64(i%9) * 384_729))
		dailyBudget := int64(380_000_000 + (int64(i%11) * 95_000_000) + (int64(i%6) * 18_473))
		fmt.Fprintf(
			w,
			"  ('%s', %s, %d, 'ACTIVE', '%s', 'ASAP', %d, 'UTC', 100000000, 3600)%s\n",
			seedCampaignUUID(i),
			sqlLiteral(seedCampaignName(i)),
			budgetLimit,
			seedCustomerUUID(i),
			dailyBudget,
			sep,
		)
	}
	fmt.Fprintln(w, `ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  current_spend = 0,
  status = 'ACTIVE',
  budget_limit = EXCLUDED.budget_limit,
  daily_budget = EXCLUDED.daily_budget,
  freq_limit = 100000000;`)
}

func writeSeedUUIDShell(w io.Writer, count int) {
	if count < 1 {
		count = 100
	}
	fmt.Fprintf(w, "AED_SEED_DEPLOYMENT_ID='%s'\n", seedDeploymentUUID())
	fmt.Fprintf(w, "AED_SEED_LICENSE_ID='%s'\n", seedLicenseRecordUUID())
	for i := 1; i <= count; i++ {
		fmt.Fprintf(w, "AED_CUSTOMER_UUID_%d='%s'\n", i, seedCustomerUUID(i))
		fmt.Fprintf(w, "AED_BRAND_UUID_%d='%s'\n", i, seedBrandUUID(i))
		fmt.Fprintf(w, "AED_CREATIVE_UUID_%d='%s'\n", i, seedCreativeUUID(i))
		fmt.Fprintf(w, "AED_CAMPAIGN_UUID_%d='%s'\n", i, seedCampaignUUID(i))
	}
}
