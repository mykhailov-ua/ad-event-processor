INSERT INTO customers (id, name, balance, currency)
VALUES ($1, $2, $3, $4)
RETURNING *;

UPDATE customers
SET balance = balance + $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

SELECT * FROM customers
WHERE id = $1
FOR UPDATE;

UPDATE campaigns
SET status = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

SELECT c.*, 
       cr.primary_a_shard, cr.primary_b_shard, cr.reserve_shard, cr.h_ema, cr.c_ema, cr.routing_epoch
FROM campaigns c
LEFT JOIN campaign_routing cr ON c.id = cr.campaign_id
WHERE c.id = $1;

INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, idempotency_hash, payment_intent_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

SELECT
  COALESCE(SUM(CASE WHEN type = 'FEE' THEN -amount ELSE 0 END), 0)::bigint AS advertiser_spend_micro,
  COALESCE(SUM(CASE WHEN type = 'rtb_cost' THEN amount ELSE 0 END), 0)::bigint AS rtb_cost_micro,
  COALESCE(SUM(CASE WHEN type = 'operator_margin' THEN amount ELSE 0 END), 0)::bigint AS operator_margin_micro,
  COALESCE(SUM(CASE WHEN type = 'publisher_payout' THEN amount ELSE 0 END), 0)::bigint AS publisher_payout_micro
FROM balance_ledger
WHERE campaign_id = $1
  AND created_at >= $2
  AND type IN ('FEE', 'rtb_cost', 'operator_margin', 'publisher_payout');

SELECT
  campaign_id,
  COALESCE(SUM(CASE WHEN type = 'FEE' THEN -amount ELSE 0 END), 0)::bigint AS advertiser_spend_micro,
  COALESCE(SUM(CASE WHEN type = 'rtb_cost' THEN amount ELSE 0 END), 0)::bigint AS rtb_cost_micro,
  COALESCE(SUM(CASE WHEN type = 'operator_margin' THEN amount ELSE 0 END), 0)::bigint AS operator_margin_micro,
  COALESCE(SUM(CASE WHEN type = 'publisher_payout' THEN amount ELSE 0 END), 0)::bigint AS publisher_payout_micro
FROM balance_ledger
WHERE campaign_id = ANY(@campaign_ids::uuid[])
  AND created_at >= @window_start
  AND type IN ('FEE', 'rtb_cost', 'operator_margin', 'publisher_payout')
GROUP BY campaign_id;

SELECT id, campaign_id, name, min_clicks, roi_floor_pct, zero_conv_streak, cost_over_revenue_threshold_bps, is_active
FROM margin_guard_policies
WHERE campaign_id = ANY($1::uuid[])
ORDER BY campaign_id, id;

SELECT DISTINCT campaign_id
FROM margin_guard_activity
WHERE campaign_id = ANY($1::uuid[])
  AND action = 'pause'
  AND placement_id = ''
  AND created_at > now() - INTERVAL '1 hour';

SELECT * FROM balance_ledger
WHERE idempotency_hash = $1;

SELECT * FROM balance_ledger
WHERE idempotency_hash = $1
FOR UPDATE;

SELECT * FROM balance_ledger
WHERE payment_intent_id = $1 AND type = 'PAYMENT_TOPUP'
FOR UPDATE;

SELECT COALESCE(SUM(ABS(amount)), 0)::bigint AS total_refunded_micro
FROM balance_ledger
WHERE payment_intent_id = $1 AND type = 'PAYMENT_REFUND';

SELECT COALESCE(SUM(ABS(amount)), 0)::bigint AS total_chargeback_micro
FROM balance_ledger
WHERE payment_intent_id = $1 AND type = 'PAYMENT_CHARGEBACK';

SELECT COALESCE(SUM(amount), 0)::bigint AS total_reversal_micro
FROM balance_ledger
WHERE payment_intent_id = $1 AND type = 'PAYMENT_CHARGEBACK_REVERSAL';

SELECT
  payment_intent_id,
  COALESCE(SUM(CASE WHEN type = 'PAYMENT_TOPUP' THEN amount ELSE 0 END), 0)::bigint AS topup_micro,
  COALESCE(SUM(CASE WHEN type = 'PAYMENT_REFUND' THEN ABS(amount) ELSE 0 END), 0)::bigint AS refund_micro,
  COALESCE(SUM(CASE WHEN type = 'PAYMENT_CHARGEBACK' THEN ABS(amount) ELSE 0 END), 0)::bigint AS chargeback_micro,
  COALESCE(SUM(CASE WHEN type = 'PAYMENT_CHARGEBACK_REVERSAL' THEN amount ELSE 0 END), 0)::bigint AS chargeback_reversal_micro,
  BOOL_OR(type = 'PAYMENT_TOPUP') AS has_topup
FROM balance_ledger
WHERE payment_intent_id = ANY($1::uuid[])
GROUP BY payment_intent_id;

SELECT id FROM balance_ledger
WHERE payment_intent_id = $1 AND type = 'PAYMENT_CHARGEBACK'
ORDER BY id;

SELECT payment_intent_id, id FROM balance_ledger
WHERE type = 'PAYMENT_CHARGEBACK'
  AND payment_intent_id = ANY($1::uuid[])
ORDER BY payment_intent_id, id;

INSERT INTO campaign_status_history (campaign_id, old_status, new_status, reason)
VALUES ($1, $2, $3, $4);

UPDATE campaigns
SET status = 'DELETED',
    deleted_at = CURRENT_TIMESTAMP
WHERE id = $1;

INSERT INTO admin_audit_log (admin_id, action, target_type, target_id, changes, metadata, is_masked)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

DELETE FROM admin_audit_log
WHERE created_at < $1;

SELECT * FROM admin_audit_log
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

SELECT COUNT(*) FROM admin_audit_log;

SELECT * FROM admin_audit_log
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

SELECT * FROM admin_audit_log
WHERE created_at >= $1 AND created_at < $2
ORDER BY created_at ASC, id ASC
LIMIT $3 OFFSET $4;

SELECT * FROM admin_audit_log
WHERE ($1::bigint = 0 OR id > $1)
ORDER BY id ASC
LIMIT $2;

SELECT * FROM balance_ledger
WHERE payment_intent_id = $1 AND type = 'PAYMENT_TOPUP'
LIMIT 1;

SELECT COUNT(*) FROM customers;

SELECT * FROM customers
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;


SELECT customer_id, COUNT(*) as active_campaigns, COALESCE(SUM(current_spend), 0)::bigint as total_spend
FROM campaigns
WHERE customer_id = ANY(@customer_ids::uuid[]) AND status = 'ACTIVE'
GROUP BY customer_id;

SELECT customer_id, id AS campaign_id
FROM campaigns
WHERE customer_id = ANY($1::uuid[])
  AND deleted_at IS NULL;

SELECT rule_id
FROM alert_rule_events
WHERE rule_id = ANY($1::uuid[])
  AND window_start = $2;

SELECT COUNT(*) FROM balance_ledger
WHERE customer_id = $1;

SELECT * FROM balance_ledger
WHERE customer_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

SELECT * FROM balance_ledger
WHERE customer_id = $1
ORDER BY id DESC
LIMIT 100;

SELECT * FROM balance_ledger
WHERE customer_id = @customer_id
  AND (@cursor_id::bigint = 0 OR id < @cursor_id::bigint)
ORDER BY id DESC
LIMIT @batch_limit;

SELECT * FROM recon_runs
ORDER BY id DESC
LIMIT $1 OFFSET $2;

SELECT COUNT(*) FROM recon_runs;

SELECT COUNT(*) FROM campaigns
WHERE (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id')::uuid)
  AND (sqlc.narg('status')::text IS NULL OR status::text = sqlc.narg('status')::text)
  AND (sqlc.narg('owner_user_id')::uuid IS NULL OR owner_user_id = sqlc.narg('owner_user_id')::uuid);

SELECT * FROM campaigns
WHERE (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id')::uuid)
  AND (sqlc.narg('status')::text IS NULL OR status::text = sqlc.narg('status')::text)
  AND (sqlc.narg('owner_user_id')::uuid IS NULL OR owner_user_id = sqlc.narg('owner_user_id')::uuid)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

SELECT COUNT(*) FROM campaign_status_history
WHERE campaign_id = $1;

SELECT * FROM campaign_status_history
WHERE campaign_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

INSERT INTO ip_blacklist (ip, reason, expires_at)
VALUES ($1, $2, $3)
ON CONFLICT (ip) DO UPDATE
    SET reason = EXCLUDED.reason,
        created_at = CURRENT_TIMESTAMP,
        expires_at = EXCLUDED.expires_at
RETURNING *;

INSERT INTO edge_block_audit (ip, reason_id, ttl, source)
VALUES ($1, $2, $3, $4)
RETURNING *;

DELETE FROM ip_blacklist
WHERE ip = $1;

SELECT ip, reason FROM ip_blacklist
WHERE expires_at IS NOT NULL AND expires_at <= NOW()
ORDER BY expires_at ASC
LIMIT $1;

SELECT COUNT(*) FROM ip_blacklist;

SELECT * FROM ip_blacklist
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

SELECT ip, reason FROM ip_blacklist;

INSERT INTO system_settings (key, value, updated_at)
VALUES ($1, $2, CURRENT_TIMESTAMP)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = CURRENT_TIMESTAMP;

SELECT key, value FROM system_settings;

SELECT value FROM system_settings WHERE key = $1;

INSERT INTO outbox_events (event_type, payload)
VALUES ($1, $2)
RETURNING *;

INSERT INTO outbox_events (event_type, payload)
SELECT unnest(@event_types::text[]), unnest(@payloads::bytea[]);

SELECT * FROM outbox_events
WHERE status = 'PENDING'
ORDER BY
  CASE event_type
    WHEN 'UPDATE_BLACKLIST' THEN 0
    WHEN 'PAUSE_CAMPAIGN' THEN 0
    WHEN 'CANCEL_CAMPAIGN' THEN 0
    WHEN 'BUDGET_FREEZE' THEN 0
    WHEN 'QUOTA_REPAIR' THEN 0
    WHEN 'ML_MODEL_VERSION' THEN 0
    ELSE 1
  END,
  created_at ASC
LIMIT $1
FOR UPDATE SKIP LOCKED;

UPDATE outbox_events
SET status = 'PROCESSED'
WHERE id = $1;

SELECT * FROM campaigns
WHERE status = 'DRAINING' AND updated_at < $1
ORDER BY updated_at ASC
LIMIT $2
FOR UPDATE SKIP LOCKED;

SELECT 
    c.id,
    COALESCE(FLOOR(EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - c.created_at)) / 86400), 0)::integer AS age_days,
    COALESCE(SUM(l.amount), 0)::bigint AS topup_sum_30d
FROM customers c
LEFT JOIN balance_ledger l ON l.customer_id = c.id 
    AND (l.type = 'TOPUP' OR l.type = 'PAYMENT_TOPUP') 
    AND l.created_at >= CURRENT_TIMESTAMP - INTERVAL '30 days'
GROUP BY c.id;

UPDATE customers
SET allowed_overdraft = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

INSERT INTO advertiser_brands (id, customer_id, name)
VALUES ($1, $2, $3)
RETURNING *;

SELECT * FROM advertiser_brands WHERE id = $1 LIMIT 1;

SELECT * FROM advertiser_brands WHERE id = $1 LIMIT 1 FOR UPDATE;

UPDATE advertiser_brands
SET freq_limit = $2,
    freq_window = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

SELECT * FROM advertiser_brands
WHERE customer_id = $1
ORDER BY created_at DESC;

SELECT 
    c.id, c.name, c.status, c.budget_limit, c.created_at, c.updated_at, c.customer_id, c.current_spend, c.deleted_at, c.pacing_mode, c.daily_budget, c.timezone, c.freq_limit, c.freq_window, c.target_countries, c.brand_id, c.brand_fcap_key,
    COALESCE(SUM(s.impressions_count), 0)::bigint AS total_impressions,
    COALESCE(SUM(s.clicks_count), 0)::bigint AS total_clicks,
    COALESCE(SUM(s.conversions_count), 0)::bigint AS total_conversions
FROM campaigns c
LEFT JOIN campaign_stats s ON c.id = s.campaign_id
WHERE c.customer_id = $1 AND c.status = 'ACTIVE'
GROUP BY c.id;

UPDATE campaigns
SET budget_limit = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

SELECT 
    c.id, c.name, c.status, c.budget_limit, c.created_at, c.updated_at, c.customer_id, c.current_spend, c.deleted_at, c.pacing_mode, c.daily_budget, c.timezone, c.freq_limit, c.freq_window, c.target_countries, c.brand_id, c.brand_fcap_key, c.daypart_hours,
    COALESCE(SUM(s.impressions_count), 0)::bigint AS total_impressions,
    COALESCE(SUM(s.clicks_count), 0)::bigint AS total_clicks,
    COALESCE(SUM(s.conversions_count), 0)::bigint AS total_conversions
FROM campaigns c
LEFT JOIN campaign_stats s ON c.id = s.campaign_id
WHERE c.status = 'ACTIVE'
GROUP BY c.id;

SELECT * FROM campaigns
WHERE id = $1
FOR UPDATE;

UPDATE campaigns
SET pacing_mode = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

UPDATE campaigns
SET name = $2,
    daily_budget = $3,
    timezone = $4,
    freq_limit = $5,
    freq_window = $6,
    target_countries = $7,
    target_url = $8,
    referrer_filter = $9,
    safe_page_url = $10,
    safe_page_enabled = $11,
    attestation_enabled = $12,
    attestation_ttl_sec = $13,
    dmr_enabled = $14,
    click_delivery = $15,
    proxy_upstream_url = $16,
    proxy_rewrite_assets = $17,
    tls_fingerprint_block_enabled = $18,
    conn_type_policy = $19,
    link_signing_enabled = $20,
    link_signing_ttl_sec = $21,
    l1_cidr_block_enabled = $22,
    l15_proxy_vpn_block_enabled = $23,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

SELECT COUNT(*)::bigint FROM events WHERE campaign_id = $1;

SELECT click_id, event_type, user_id, payload, ip_address, user_agent, created_at
FROM events
WHERE campaign_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

UPDATE campaigns
SET fraud_threshold_pass = $2,
    fraud_threshold_suspect = $3,
    fraud_threshold_ivt = $4,
    fraud_threshold_block = $5,
    ghost_ivt_enabled = $6,
    behavior_flags = $7,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;


SELECT 
    campaign_id,
    COALESCE(SUM(CASE WHEN amount < 0 THEN -amount ELSE 0 END), 0)::bigint AS total_spent_micro
FROM balance_ledger
WHERE created_at >= $1 
  AND created_at < $2
  AND (type = 'FEE' OR type = 'RECONCILIATION_ADJUST' OR type = 'REFUND')
GROUP BY campaign_id;

SELECT
    bl.campaign_id,
    c.customer_id,
    COALESCE(SUM(CASE WHEN bl.amount < 0 THEN -bl.amount ELSE 0 END), 0)::bigint AS total_spent_micro
FROM balance_ledger bl
INNER JOIN campaigns c ON c.id = bl.campaign_id
WHERE bl.created_at >= $1
  AND bl.created_at < $2
  AND (bl.type = 'FEE' OR bl.type = 'RECONCILIATION_ADJUST' OR bl.type = 'REFUND')
GROUP BY bl.campaign_id, c.customer_id;

INSERT INTO recon_runs (period_start, period_end, status)
VALUES ($1, $2, 'PENDING')
RETURNING *;

UPDATE recon_runs
SET status = $2,
    total_delta = $3,
    campaigns_checked = $4,
    discrepancies_found = $5,
    completed_at = NOW()
WHERE id = $1;

INSERT INTO recon_discrepancies (
    run_id, campaign_id, customer_id, expected_spend, actual_spend, delta, redis_adjusted
) VALUES ($1, $2, $3, $4, $5, $6, $7);

SELECT COALESCE(MAX(ABS(delta)), 0)::bigint AS max_lag_micro
FROM recon_discrepancies
WHERE customer_id = $1
  AND created_at >= CURRENT_TIMESTAMP - INTERVAL '24 hours';

SELECT
    customer_id,
    COALESCE(MAX(ABS(delta)), 0)::bigint AS max_lag_micro
FROM recon_discrepancies
WHERE created_at >= CURRENT_TIMESTAMP - INTERVAL '24 hours'
GROUP BY customer_id;

SELECT
    COALESCE(SUM(impressions_count), 0)::bigint AS impressions,
    COALESCE(SUM(clicks_count), 0)::bigint AS clicks,
    COALESCE(SUM(conversions_count), 0)::bigint AS conversions
FROM campaign_stats
WHERE campaign_id = @campaign_id
  AND date >= @from_date::date
  AND date <= @to_date::date;

SELECT
    cs.campaign_id,
    COALESCE(SUM(cs.impressions_count), 0)::bigint AS impressions,
    COALESCE(SUM(cs.clicks_count), 0)::bigint AS clicks,
    COALESCE(SUM(cs.conversions_count), 0)::bigint AS conversions
FROM campaign_stats cs
INNER JOIN campaigns c ON c.id = cs.campaign_id
WHERE c.customer_id = @customer_id
  AND cs.date >= @from_date::date
  AND cs.date <= @to_date::date
GROUP BY cs.campaign_id;

SELECT * FROM sellers ORDER BY seller_id;

SELECT * FROM sellers WHERE id = $1;

INSERT INTO sellers (seller_id, domain, seller_type, name, is_confidential)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

UPDATE sellers
SET seller_id = $2,
    domain = $3,
    seller_type = $4,
    name = $5,
    is_confidential = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

DELETE FROM sellers WHERE id = $1;

SELECT * FROM ads_txt_entries ORDER BY sort_order, id;

SELECT * FROM ads_txt_entries WHERE id = $1;

INSERT INTO ads_txt_entries (domain, publisher_account_id, relationship, cert_authority_id, sort_order)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

UPDATE ads_txt_entries
SET domain = $2,
    publisher_account_id = $3,
    relationship = $4,
    cert_authority_id = $5,
    sort_order = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

DELETE FROM ads_txt_entries WHERE id = $1;

UPDATE campaigns
SET supply_chain_nodes = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

SELECT * FROM rtb_deals ORDER BY deal_id;

SELECT * FROM rtb_deals WHERE id = $1;

SELECT * FROM rtb_deals WHERE deal_id = $1;

INSERT INTO rtb_deals (deal_id, floor_micro, geo_mask, cat_mask, pacing, customer_id, seats)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

UPDATE rtb_deals
SET deal_id = $2,
    floor_micro = $3,
    geo_mask = $4,
    cat_mask = $5,
    pacing = $6,
    customer_id = $7,
    seats = $8,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

DELETE FROM rtb_deals WHERE id = $1;

INSERT INTO rtb_floor_suggestions (
    placement_id, deal_id, current_floor_micro, suggested_floor_micro,
    win_rate, sample_n, floor_bucket_micro, computed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (placement_id) DO UPDATE SET
    deal_id = EXCLUDED.deal_id,
    current_floor_micro = EXCLUDED.current_floor_micro,
    suggested_floor_micro = EXCLUDED.suggested_floor_micro,
    win_rate = EXCLUDED.win_rate,
    sample_n = EXCLUDED.sample_n,
    floor_bucket_micro = EXCLUDED.floor_bucket_micro,
    computed_at = EXCLUDED.computed_at;

SELECT placement_id, deal_id, current_floor_micro, suggested_floor_micro,
       win_rate, sample_n, floor_bucket_micro, computed_at
FROM rtb_floor_suggestions
ORDER BY placement_id;

SELECT placement_id, deal_id, current_floor_micro, suggested_floor_micro,
       win_rate, sample_n, floor_bucket_micro, computed_at
FROM rtb_floor_suggestions
WHERE placement_id = ANY($1::text[])
ORDER BY placement_id;

INSERT INTO campaign_shard_assignment (
    campaign_id, primary_a_shard, primary_b_shard, reserve_shard, h_ema, c_ema, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (campaign_id) DO UPDATE SET
    primary_a_shard = EXCLUDED.primary_a_shard,
    primary_b_shard = EXCLUDED.primary_b_shard,
    reserve_shard = EXCLUDED.reserve_shard,
    h_ema = EXCLUDED.h_ema,
    c_ema = EXCLUDED.c_ema,
    updated_at = NOW()
RETURNING *;

SELECT * FROM campaign_shard_assignment
WHERE campaign_id = $1;

DELETE FROM campaign_shard_assignment
WHERE campaign_id = $1;

SELECT *
FROM ctv_gtax_settlements
WHERE settlement_id = $1;

INSERT INTO ctv_gtax_settlements (
  settlement_id, customer_id, campaign_id, spend_micro, tax_micro, fee_ledger_id, tax_ledger_id
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

SELECT id, name, active, salt, variants, created_at, updated_at
FROM experiment_cohorts
WHERE active = TRUE
ORDER BY name;

INSERT INTO experiment_cohorts (id, name, active, salt, variants)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    active = EXCLUDED.active,
    salt = EXCLUDED.salt,
    variants = EXCLUDED.variants,
    updated_at = now()
RETURNING *;

SELECT id, name, active, salt, variants, created_at, updated_at
FROM experiment_cohorts
WHERE id = $1;

