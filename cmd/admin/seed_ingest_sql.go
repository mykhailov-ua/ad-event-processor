package main

import (
	"fmt"
	"io"
	"strings"
)

func writeSeedSQL(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func writeSeedSQLLine(w io.Writer, line string) {
	_, _ = fmt.Fprintln(w, line)
}

func sqlLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func writeSeedIngestSQL(w io.Writer, count int) {
	if count < 1 {
		count = 100
	}

	writeSeedSQLLine(w, "INSERT INTO customers (id, name, balance, currency, allowed_overdraft)")
	writeSeedSQLLine(w, "VALUES")
	for i := 1; i <= count; i++ {
		sep := ","
		if i == count {
			sep = ""
		}
		writeSeedSQL(
			w,
			"  ('%s', %s, %d, 'USD', 0)%s\n",
			seedCustomerUUID(i),
			sqlLiteral(seedCustomerName(i)),
			seedCustomerBalanceMicro(i),
			sep,
		)
	}
	writeSeedSQLLine(w, "ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, balance = EXCLUDED.balance;")
	writeSeedSQLLine(w, "")

	writeSeedSQLLine(w, "INSERT INTO advertiser_brands (id, customer_id, name)")
	writeSeedSQLLine(w, "VALUES")
	for i := 1; i <= count; i++ {
		sep := ","
		if i == count {
			sep = ""
		}
		writeSeedSQL(
			w,
			"  ('%s', '%s', %s)%s\n",
			seedBrandUUID(i),
			seedCustomerUUID(i),
			sqlLiteral(seedBrandDisplayName(i)),
			sep,
		)
	}
	writeSeedSQLLine(w, "ON CONFLICT (id) DO NOTHING;")
	writeSeedSQLLine(w, "")

	writeSeedSQLLine(w, "INSERT INTO brand_creatives (id, brand_id, name, landing_url, weight, status)")
	writeSeedSQLLine(w, "VALUES")
	for i := 1; i <= count; i++ {
		sep := ","
		if i == count {
			sep = ""
		}
		writeSeedSQL(
			w,
			"  ('%s', '%s', %s, 'https://trk.horizon-media.io/landing?cid={click_id}', %d, 'ACTIVE')%s\n",
			seedCreativeUUID(i),
			seedBrandUUID(i),
			sqlLiteral(seedCreativeDisplayName(i)),
			97+(i%13),
			sep,
		)
	}
	writeSeedSQLLine(w, `ON CONFLICT (brand_id, name) DO UPDATE SET
  landing_url = EXCLUDED.landing_url,
  status = 'ACTIVE',
  updated_at = NOW();`)
	writeSeedSQLLine(w, "")

	writeSeedSQLLine(w, "INSERT INTO campaigns (id, name, budget_limit, status, customer_id, pacing_mode, daily_budget, timezone, freq_limit, freq_window, brand_id, target_url, target_countries)")
	writeSeedSQLLine(w, "VALUES")
	for i := 1; i <= count; i++ {
		sep := ","
		if i == count {
			sep = ""
		}
		budgetLimit := int64(4_200_000_000 + (int64(i%17) * 650_000_000) + (int64(i%9) * 384_729))
		dailyBudget := int64(380_000_000 + (int64(i%11) * 95_000_000) + (int64(i%6) * 18_473))
		targetCountries := seedUIDemoTargetCountries(i)
		writeSeedSQL(
			w,
			"  ('%s', %s, %d, 'ACTIVE', '%s', 'ASAP', %d, 'UTC', 100000000, 3600, '%s', 'https://trk.horizon-media.io/landing?cid={click_id}', %s)%s\n",
			seedCampaignUUID(i),
			sqlLiteral(seedCampaignName(i)),
			budgetLimit,
			seedCustomerUUID(i),
			dailyBudget,
			seedBrandUUID(i),
			sqlLiteral(formatPostgresTextArray(targetCountries)),
			sep,
		)
	}
	writeSeedSQLLine(w, `ON CONFLICT (id) DO UPDATE SET
  current_spend = 0,
  status = 'ACTIVE',
  budget_limit = EXCLUDED.budget_limit,
  brand_id = EXCLUDED.brand_id,
  target_url = EXCLUDED.target_url,
  target_countries = EXCLUDED.target_countries;`)
	writeSeedSQLLine(w, "")

	writeSeedSQL(w, `INSERT INTO billing.license_status (
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

	writeSeedSQLLine(w, "INSERT INTO customers (id, name, balance, currency, allowed_overdraft)")
	writeSeedSQLLine(w, "VALUES")
	for i := 1; i <= count; i++ {
		sep := ","
		if i == count {
			sep = ""
		}
		writeSeedSQL(
			w,
			"  ('%s', %s, %d, 'USD', 0)%s\n",
			seedCustomerUUID(i),
			sqlLiteral(seedCustomerName(i)),
			seedCustomerBalanceMicro(i),
			sep,
		)
	}
	writeSeedSQLLine(w, `ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  balance = EXCLUDED.balance;`)
	writeSeedSQLLine(w, "")

	writeSeedSQLLine(w, "INSERT INTO campaigns (id, name, budget_limit, status, customer_id, pacing_mode, daily_budget, timezone, freq_limit, freq_window)")
	writeSeedSQLLine(w, "VALUES")
	for i := 1; i <= count; i++ {
		sep := ","
		if i == count {
			sep = ""
		}
		budgetLimit := int64(4_200_000_000 + (int64(i%17) * 650_000_000) + (int64(i%9) * 384_729))
		dailyBudget := int64(380_000_000 + (int64(i%11) * 95_000_000) + (int64(i%6) * 18_473))
		writeSeedSQL(
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
	writeSeedSQLLine(w, `ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  current_spend = 0,
  status = 'ACTIVE',
  budget_limit = EXCLUDED.budget_limit,
  daily_budget = EXCLUDED.daily_budget,
  freq_limit = 100000000;`)
}

func writeLoadTestStackSeedSQL(w io.Writer, count int) {
	if count < 1 {
		count = 100
	}

	writeSeedSQLLine(w, "TRUNCATE TABLE events CASCADE;")
	writeSeedSQLLine(w, "TRUNCATE TABLE campaign_stats CASCADE;")
	writeSeedSQLLine(w, "TRUNCATE TABLE campaigns CASCADE;")
	writeSeedSQLLine(w, "")

	writeSeedSQLLine(w, "INSERT INTO customers (id, name, balance, currency, allowed_overdraft)")
	writeSeedSQLLine(w, "VALUES")
	for i := 1; i <= count; i++ {
		sep := ","
		if i == count {
			sep = ""
		}
		writeSeedSQL(
			w,
			"  ('%s', %s, %d, 'USD', 0)%s\n",
			loadTestSequentialUUID(i),
			sqlLiteral(seedCustomerName(i)),
			seedCustomerBalanceMicro(i),
			sep,
		)
	}
	writeSeedSQLLine(w, `ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  balance = EXCLUDED.balance;`)
	writeSeedSQLLine(w, "")

	writeSeedSQLLine(w, "INSERT INTO campaigns (id, name, budget_limit, status, customer_id, pacing_mode, daily_budget, timezone, freq_limit, freq_window)")
	writeSeedSQLLine(w, "VALUES")
	for i := 1; i <= count; i++ {
		sep := ","
		if i == count {
			sep = ""
		}
		budgetLimit := int64(4_200_000_000 + (int64(i%17) * 650_000_000) + (int64(i%9) * 384_729))
		dailyBudget := int64(380_000_000 + (int64(i%11) * 95_000_000) + (int64(i%6) * 18_473))
		writeSeedSQL(
			w,
			"  ('%s', %s, %d, 'ACTIVE', '%s', 'ASAP', %d, 'UTC', 100000000, 3600)%s\n",
			loadTestSequentialUUID(i),
			sqlLiteral(seedCampaignName(i)),
			budgetLimit,
			loadTestSequentialUUID(i),
			dailyBudget,
			sep,
		)
	}
	writeSeedSQLLine(w, `ON CONFLICT (id) DO UPDATE SET
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
	writeSeedSQL(w, "AED_SEED_DEPLOYMENT_ID='%s'\n", seedDeploymentUUID())
	writeSeedSQL(w, "AED_SEED_LICENSE_ID='%s'\n", seedLicenseRecordUUID())
	for i := 1; i <= count; i++ {
		writeSeedSQL(w, "AED_CUSTOMER_UUID_%d='%s'\n", i, seedCustomerUUID(i))
		writeSeedSQL(w, "AED_BRAND_UUID_%d='%s'\n", i, seedBrandUUID(i))
		writeSeedSQL(w, "AED_CREATIVE_UUID_%d='%s'\n", i, seedCreativeUUID(i))
		writeSeedSQL(w, "AED_CAMPAIGN_UUID_%d='%s'\n", i, seedCampaignUUID(i))
	}
}
