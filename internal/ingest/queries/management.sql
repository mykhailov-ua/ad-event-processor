-- name: CreateCustomer :one
INSERT INTO customers (id, name, balance, currency)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateCustomerBalanceManagement :one
UPDATE customers
SET balance = balance + $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: GetCustomerForUpdate :one
SELECT * FROM customers
WHERE id = $1
FOR UPDATE;

-- name: UpdateCampaignStatus :one
UPDATE campaigns
SET status = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: GetCampaignFull :one
SELECT c.*, 
       cr.primary_a_shard, cr.primary_b_shard, cr.reserve_shard, cr.h_ema, cr.c_ema, cr.routing_epoch
FROM campaigns c
LEFT JOIN campaign_routing cr ON c.id = cr.campaign_id
WHERE c.id = $1;

-- name: ListCampaignsByIDs :many
SELECT * FROM campaigns WHERE id = ANY($1::uuid[]);

-- name: CreateLedgerEntry :one
INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, idempotency_hash, payment_intent_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: SumCampaignMarginWindow :one
SELECT
  COALESCE(SUM(CASE WHEN type = 'FEE' THEN -amount ELSE 0 END), 0)::bigint AS advertiser_spend_micro,
  COALESCE(SUM(CASE WHEN type = 'rtb_cost' THEN amount ELSE 0 END), 0)::bigint AS rtb_cost_micro,
  COALESCE(SUM(CASE WHEN type = 'operator_margin' THEN amount ELSE 0 END), 0)::bigint AS operator_margin_micro,
  COALESCE(SUM(CASE WHEN type = 'publisher_payout' THEN amount ELSE 0 END), 0)::bigint AS publisher_payout_micro
FROM balance_ledger
WHERE campaign_id = $1
  AND created_at >= $2
  AND type IN ('FEE', 'rtb_cost', 'operator_margin', 'publisher_payout');

-- name: SumCampaignMarginWindowByCampaignIDs :many
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

-- name: SumCampaignMarginWindowByCampaignIDsInRange :many
SELECT
  campaign_id,
  COALESCE(SUM(CASE WHEN type = 'FEE' THEN -amount ELSE 0 END), 0)::bigint AS advertiser_spend_micro,
  COALESCE(SUM(CASE WHEN type = 'rtb_cost' THEN amount ELSE 0 END), 0)::bigint AS rtb_cost_micro,
  COALESCE(SUM(CASE WHEN type = 'operator_margin' THEN amount ELSE 0 END), 0)::bigint AS operator_margin_micro,
  COALESCE(SUM(CASE WHEN type = 'publisher_payout' THEN amount ELSE 0 END), 0)::bigint AS publisher_payout_micro
FROM balance_ledger
WHERE campaign_id = ANY(@campaign_ids::uuid[])
  AND created_at >= @window_start
  AND created_at < @window_end
  AND type IN ('FEE', 'rtb_cost', 'operator_margin', 'publisher_payout')
GROUP BY campaign_id;

-- name: ListMarginGuardPoliciesByCampaignIDs :many
SELECT id, campaign_id, name, min_clicks, roi_floor_pct, zero_conv_streak, cost_over_revenue_threshold_bps, is_active
FROM margin_guard_policies
WHERE campaign_id = ANY($1::uuid[])
ORDER BY campaign_id, id;

-- name: ListRecentMarginGuardPausesByCampaigns :many
SELECT DISTINCT campaign_id
FROM margin_guard_activity
WHERE campaign_id = ANY($1::uuid[])
  AND action = 'pause'
  AND placement_id = ''
  AND created_at > now() - INTERVAL '1 hour';

-- name: GetLedgerByHash :one
SELECT * FROM balance_ledger
WHERE idempotency_hash = $1;

-- name: GetLedgerByHashForUpdate :one
SELECT * FROM balance_ledger
WHERE idempotency_hash = $1
FOR UPDATE;

-- name: GetLedgerByPaymentIntentForUpdate :one
SELECT * FROM balance_ledger
WHERE payment_intent_id = $1 AND type = 'PAYMENT_TOPUP'
FOR UPDATE;

-- name: SumPaymentRefundAmountForIntent :one
SELECT COALESCE(SUM(ABS(amount)), 0)::bigint AS total_refunded_micro
FROM balance_ledger
WHERE payment_intent_id = $1 AND type = 'PAYMENT_REFUND';

-- name: SumPaymentChargebackAmountForIntent :one
SELECT COALESCE(SUM(ABS(amount)), 0)::bigint AS total_chargeback_micro
FROM balance_ledger
WHERE payment_intent_id = $1 AND type = 'PAYMENT_CHARGEBACK';

-- name: SumPaymentChargebackReversalAmountForIntent :one
SELECT COALESCE(SUM(amount), 0)::bigint AS total_reversal_micro
FROM balance_ledger
WHERE payment_intent_id = $1 AND type = 'PAYMENT_CHARGEBACK_REVERSAL';

-- name: SumPaymentLedgerTotalsByIntentIDs :many
SELECT
    payment_intent_id,
    COALESCE(SUM(CASE WHEN type = 'PAYMENT_TOPUP' THEN amount ELSE 0 END), 0)::bigint AS topup_micro,
    BOOL_OR(type = 'PAYMENT_TOPUP') AS has_topup,
    COALESCE(SUM(CASE WHEN type = 'PAYMENT_REFUND' THEN ABS(amount) ELSE 0 END), 0)::bigint AS refund_micro,
    COALESCE(SUM(CASE WHEN type = 'PAYMENT_CHARGEBACK' THEN ABS(amount) ELSE 0 END), 0)::bigint AS chargeback_micro,
    COALESCE(SUM(CASE WHEN type = 'PAYMENT_CHARGEBACK_REVERSAL' THEN amount ELSE 0 END), 0)::bigint AS chargeback_reversal_micro
FROM balance_ledger
WHERE payment_intent_id = ANY($1::uuid[])
GROUP BY payment_intent_id;

-- name: ListLedgerChargebackEntryIDs :many
SELECT id FROM balance_ledger
WHERE payment_intent_id = $1 AND type = 'PAYMENT_CHARGEBACK'
ORDER BY id;

-- name: ListLedgerChargebackEntryIDsByIntents :many
SELECT payment_intent_id, id FROM balance_ledger
WHERE type = 'PAYMENT_CHARGEBACK'
  AND payment_intent_id = ANY($1::uuid[])
ORDER BY payment_intent_id, id;

-- name: CreateStatusHistory :exec
INSERT INTO campaign_status_history (campaign_id, old_status, new_status, reason)
VALUES ($1, $2, $3, $4);

-- name: SoftDeleteCampaign :exec
UPDATE campaigns
SET status = 'DELETED',
    deleted_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: CreateAuditLog :one
INSERT INTO admin_audit_log (admin_id, action, target_type, target_id, changes, metadata, is_masked)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: CleanupAuditLogs :exec
DELETE FROM admin_audit_log
WHERE created_at < $1;

-- name: ListAuditLogs :many
SELECT * FROM admin_audit_log
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAuditLogs :one
SELECT COUNT(*) FROM admin_audit_log;

-- name: ListAuditPaginated :many
SELECT * FROM admin_audit_log
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListAuditLogsInRange :many
SELECT * FROM admin_audit_log
WHERE created_at >= $1 AND created_at < $2
ORDER BY created_at ASC, id ASC
LIMIT $3 OFFSET $4;

-- name: ListAuditLogsExport :many
SELECT * FROM admin_audit_log
WHERE ($1::bigint = 0 OR id > $1)
ORDER BY id ASC
LIMIT $2;

-- name: GetLedgerByPaymentIntent :one
SELECT * FROM balance_ledger
WHERE payment_intent_id = $1 AND type = 'PAYMENT_TOPUP'
LIMIT 1;

-- name: CountCustomers :one
SELECT COUNT(*) FROM customers;

-- name: ListCustomers :many
SELECT * FROM customers
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;


-- name: GetCustomerStats :many
SELECT customer_id, COUNT(*) as active_campaigns, COALESCE(SUM(current_spend), 0)::bigint as total_spend
FROM campaigns
WHERE customer_id = ANY(@customer_ids::uuid[]) AND status = 'ACTIVE'
GROUP BY customer_id;

-- name: CountCustomerLedger :one
SELECT COUNT(*) FROM balance_ledger
WHERE customer_id = $1;

-- name: ListCustomerLedger :many
SELECT * FROM balance_ledger
WHERE customer_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListCustomerLedgerByIDDesc :many
SELECT * FROM balance_ledger
WHERE customer_id = $1
ORDER BY id DESC
LIMIT 100;

-- name: ListCustomerLedgerExport :many
SELECT * FROM balance_ledger
WHERE customer_id = @customer_id
  AND (@cursor_id::bigint = 0 OR id < @cursor_id::bigint)
ORDER BY id DESC
LIMIT @batch_limit;

-- name: ListManagementReconRuns :many
SELECT * FROM recon_runs
ORDER BY id DESC
LIMIT $1 OFFSET $2;

-- name: CountManagementReconRuns :one
SELECT COUNT(*) FROM recon_runs;

-- name: CountCampaigns :one
SELECT COUNT(*) FROM campaigns
WHERE deleted_at IS NULL
  AND (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id')::uuid)
  AND (sqlc.narg('status')::text IS NULL OR status::text = sqlc.narg('status')::text)
  AND (sqlc.narg('owner_user_id')::uuid IS NULL OR owner_user_id = sqlc.narg('owner_user_id')::uuid)
  AND (sqlc.narg('target_country')::text IS NULL OR sqlc.narg('target_country')::text = ANY(target_countries))
  AND (sqlc.narg('budget_min_micro')::bigint IS NULL OR budget_limit >= sqlc.narg('budget_min_micro')::bigint)
  AND (sqlc.narg('budget_max_micro')::bigint IS NULL OR budget_limit <= sqlc.narg('budget_max_micro')::bigint)
  AND (
    sqlc.narg('search_query')::text IS NULL
    OR btrim(sqlc.narg('search_query')::text) = ''
    OR name ILIKE '%' || btrim(sqlc.narg('search_query')::text) || '%'
    OR id::text ILIKE '%' || btrim(sqlc.narg('search_query')::text) || '%'
  )
  AND (
    sqlc.narg('pacing_mode')::text IS NULL
    OR btrim(sqlc.narg('pacing_mode')::text) = ''
    OR pacing_mode::text ILIKE btrim(sqlc.narg('pacing_mode')::text)
  );

-- name: ListCampaignTargetCountries :many
SELECT DISTINCT country_code::text
FROM campaigns c
CROSS JOIN LATERAL unnest(c.target_countries) AS country_code
WHERE c.deleted_at IS NULL
  AND country_code IS NOT NULL
  AND btrim(country_code) <> ''
  AND (sqlc.narg('customer_id')::uuid IS NULL OR c.customer_id = sqlc.narg('customer_id')::uuid)
ORDER BY country_code ASC;

-- name: ListCampaignListOwners :many
SELECT DISTINCT c.owner_user_id::text
FROM campaigns c
WHERE c.deleted_at IS NULL
  AND c.owner_user_id IS NOT NULL
  AND (sqlc.narg('customer_id')::uuid IS NULL OR c.customer_id = sqlc.narg('customer_id')::uuid)
ORDER BY 1 ASC;

-- name: CountCampaignsStatusTotals :many
SELECT status::text AS status, COUNT(*)::bigint AS count
FROM campaigns
WHERE deleted_at IS NULL
  AND (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id')::uuid)
  AND (sqlc.narg('owner_user_id')::uuid IS NULL OR owner_user_id = sqlc.narg('owner_user_id')::uuid)
  AND (sqlc.narg('target_country')::text IS NULL OR sqlc.narg('target_country')::text = ANY(target_countries))
  AND (sqlc.narg('budget_min_micro')::bigint IS NULL OR budget_limit >= sqlc.narg('budget_min_micro')::bigint)
  AND (sqlc.narg('budget_max_micro')::bigint IS NULL OR budget_limit <= sqlc.narg('budget_max_micro')::bigint)
  AND (
    sqlc.narg('search_query')::text IS NULL
    OR btrim(sqlc.narg('search_query')::text) = ''
    OR name ILIKE '%' || btrim(sqlc.narg('search_query')::text) || '%'
    OR id::text ILIKE '%' || btrim(sqlc.narg('search_query')::text) || '%'
  )
  AND (
    sqlc.narg('pacing_mode')::text IS NULL
    OR btrim(sqlc.narg('pacing_mode')::text) = ''
    OR pacing_mode::text ILIKE btrim(sqlc.narg('pacing_mode')::text)
  )
GROUP BY status;

-- name: CountCampaignFlowsForFilter :one
SELECT COUNT(*)::bigint FROM campaigns
WHERE deleted_at IS NULL
  AND flow_id IS NOT NULL
  AND (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id')::uuid)
  AND (sqlc.narg('status')::text IS NULL OR status::text = sqlc.narg('status')::text)
  AND (sqlc.narg('owner_user_id')::uuid IS NULL OR owner_user_id = sqlc.narg('owner_user_id')::uuid)
  AND (sqlc.narg('target_country')::text IS NULL OR sqlc.narg('target_country')::text = ANY(target_countries))
  AND (sqlc.narg('budget_min_micro')::bigint IS NULL OR budget_limit >= sqlc.narg('budget_min_micro')::bigint)
  AND (sqlc.narg('budget_max_micro')::bigint IS NULL OR budget_limit <= sqlc.narg('budget_max_micro')::bigint)
  AND (
    sqlc.narg('search_query')::text IS NULL
    OR btrim(sqlc.narg('search_query')::text) = ''
    OR name ILIKE '%' || btrim(sqlc.narg('search_query')::text) || '%'
    OR id::text ILIKE '%' || btrim(sqlc.narg('search_query')::text) || '%'
  )
  AND (
    sqlc.narg('pacing_mode')::text IS NULL
    OR btrim(sqlc.narg('pacing_mode')::text) = ''
    OR pacing_mode::text ILIKE btrim(sqlc.narg('pacing_mode')::text)
  );

-- name: ListCampaignListKeysForFilter :many
SELECT id, name FROM campaigns
WHERE deleted_at IS NULL
  AND (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id')::uuid)
  AND (sqlc.narg('status')::text IS NULL OR status::text = sqlc.narg('status')::text)
  AND (sqlc.narg('owner_user_id')::uuid IS NULL OR owner_user_id = sqlc.narg('owner_user_id')::uuid)
  AND (sqlc.narg('target_country')::text IS NULL OR sqlc.narg('target_country')::text = ANY(target_countries))
  AND (sqlc.narg('budget_min_micro')::bigint IS NULL OR budget_limit >= sqlc.narg('budget_min_micro')::bigint)
  AND (sqlc.narg('budget_max_micro')::bigint IS NULL OR budget_limit <= sqlc.narg('budget_max_micro')::bigint)
  AND (
    sqlc.narg('search_query')::text IS NULL
    OR btrim(sqlc.narg('search_query')::text) = ''
    OR name ILIKE '%' || btrim(sqlc.narg('search_query')::text) || '%'
    OR id::text ILIKE '%' || btrim(sqlc.narg('search_query')::text) || '%'
  )
  AND (
    sqlc.narg('pacing_mode')::text IS NULL
    OR btrim(sqlc.narg('pacing_mode')::text) = ''
    OR pacing_mode::text ILIKE btrim(sqlc.narg('pacing_mode')::text)
  );

-- name: ListCampaigns :many
SELECT campaigns.* FROM campaigns
LEFT JOIN customers customer_group ON customer_group.id = campaigns.customer_id
WHERE campaigns.deleted_at IS NULL
  AND (sqlc.narg('customer_id')::uuid IS NULL OR campaigns.customer_id = sqlc.narg('customer_id')::uuid)
  AND (sqlc.narg('status')::text IS NULL OR campaigns.status::text = sqlc.narg('status')::text)
  AND (sqlc.narg('owner_user_id')::uuid IS NULL OR campaigns.owner_user_id = sqlc.narg('owner_user_id')::uuid)
  AND (sqlc.narg('target_country')::text IS NULL OR sqlc.narg('target_country')::text = ANY(campaigns.target_countries))
  AND (sqlc.narg('budget_min_micro')::bigint IS NULL OR campaigns.budget_limit >= sqlc.narg('budget_min_micro')::bigint)
  AND (sqlc.narg('budget_max_micro')::bigint IS NULL OR campaigns.budget_limit <= sqlc.narg('budget_max_micro')::bigint)
  AND (
    sqlc.narg('search_query')::text IS NULL
    OR btrim(sqlc.narg('search_query')::text) = ''
    OR campaigns.name ILIKE '%' || btrim(sqlc.narg('search_query')::text) || '%'
    OR campaigns.id::text ILIKE '%' || btrim(sqlc.narg('search_query')::text) || '%'
  )
  AND (
    sqlc.narg('pacing_mode')::text IS NULL
    OR btrim(sqlc.narg('pacing_mode')::text) = ''
    OR campaigns.pacing_mode::text ILIKE btrim(sqlc.narg('pacing_mode')::text)
  )
ORDER BY
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'name' AND COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN lower(campaigns.name) END) DESC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'name' AND NOT COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN lower(campaigns.name) END) ASC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'spend' AND COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN campaigns.current_spend END) DESC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'spend' AND NOT COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN campaigns.current_spend END) ASC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'budget_limit' AND COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN campaigns.budget_limit END) DESC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'budget_limit' AND NOT COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN campaigns.budget_limit END) ASC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'budget_pct' AND COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN CASE WHEN campaigns.budget_limit > 0 THEN campaigns.current_spend::numeric / campaigns.budget_limit::numeric ELSE 0 END END) DESC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'budget_pct' AND NOT COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN CASE WHEN campaigns.budget_limit > 0 THEN campaigns.current_spend::numeric / campaigns.budget_limit::numeric ELSE 0 END END) ASC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'status' AND COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN campaigns.status::text END) DESC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'status' AND NOT COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN campaigns.status::text END) ASC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'owner' AND COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN campaigns.owner_user_id END) DESC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'owner' AND NOT COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN campaigns.owner_user_id END) ASC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'flow' AND COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN campaigns.flow_id END) DESC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'flow' AND NOT COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN campaigns.flow_id END) ASC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'group' AND COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN lower(customer_group.name) END) DESC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'group' AND NOT COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN lower(customer_group.name) END) ASC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'countries' AND COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN array_to_string(campaigns.target_countries, ',') END) DESC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'countries' AND NOT COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN array_to_string(campaigns.target_countries, ',') END) ASC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'tags' AND COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN lower(campaigns.name) END) DESC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'tags' AND NOT COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN lower(campaigns.name) END) ASC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'updated_at' AND COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN campaigns.updated_at END) DESC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'updated_at' AND NOT COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN campaigns.updated_at END) ASC NULLS LAST,
  campaigns.updated_at DESC
LIMIT $1 OFFSET $2;

-- name: ListCampaignsSortedByStats :many
SELECT campaigns.* FROM campaigns
LEFT JOIN (
  SELECT
    cs.campaign_id,
    COALESCE(SUM(cs.impressions_count), 0)::bigint AS stat_impressions,
    COALESCE(SUM(cs.clicks_count), 0)::bigint AS stat_clicks,
    COALESCE(SUM(cs.conversions_count), 0)::bigint AS stat_conversions
  FROM campaign_stats cs
  INNER JOIN (
    SELECT id FROM campaigns
    WHERE deleted_at IS NULL
      AND (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id')::uuid)
      AND (sqlc.narg('status')::text IS NULL OR status::text = sqlc.narg('status')::text)
      AND (sqlc.narg('owner_user_id')::uuid IS NULL OR owner_user_id = sqlc.narg('owner_user_id')::uuid)
      AND (sqlc.narg('target_country')::text IS NULL OR sqlc.narg('target_country')::text = ANY(target_countries))
      AND (sqlc.narg('budget_min_micro')::bigint IS NULL OR budget_limit >= sqlc.narg('budget_min_micro')::bigint)
      AND (sqlc.narg('budget_max_micro')::bigint IS NULL OR budget_limit <= sqlc.narg('budget_max_micro')::bigint)
      AND (
        sqlc.narg('search_query')::text IS NULL
        OR btrim(sqlc.narg('search_query')::text) = ''
        OR name ILIKE '%' || btrim(sqlc.narg('search_query')::text) || '%'
        OR id::text ILIKE '%' || btrim(sqlc.narg('search_query')::text) || '%'
      )
      AND (
        sqlc.narg('pacing_mode')::text IS NULL
        OR btrim(sqlc.narg('pacing_mode')::text) = ''
        OR pacing_mode::text ILIKE btrim(sqlc.narg('pacing_mode')::text)
      )
  ) filtered ON filtered.id = cs.campaign_id
  WHERE cs.date >= sqlc.narg('stats_from')::date
    AND cs.date <= sqlc.narg('stats_to')::date
  GROUP BY cs.campaign_id
) stats ON stats.campaign_id = campaigns.id
WHERE deleted_at IS NULL
  AND (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id')::uuid)
  AND (sqlc.narg('status')::text IS NULL OR status::text = sqlc.narg('status')::text)
  AND (sqlc.narg('owner_user_id')::uuid IS NULL OR owner_user_id = sqlc.narg('owner_user_id')::uuid)
  AND (sqlc.narg('target_country')::text IS NULL OR sqlc.narg('target_country')::text = ANY(target_countries))
  AND (sqlc.narg('budget_min_micro')::bigint IS NULL OR budget_limit >= sqlc.narg('budget_min_micro')::bigint)
  AND (sqlc.narg('budget_max_micro')::bigint IS NULL OR budget_limit <= sqlc.narg('budget_max_micro')::bigint)
  AND (
    sqlc.narg('search_query')::text IS NULL
    OR btrim(sqlc.narg('search_query')::text) = ''
    OR name ILIKE '%' || btrim(sqlc.narg('search_query')::text) || '%'
    OR id::text ILIKE '%' || btrim(sqlc.narg('search_query')::text) || '%'
  )
  AND (
    sqlc.narg('pacing_mode')::text IS NULL
    OR btrim(sqlc.narg('pacing_mode')::text) = ''
    OR pacing_mode::text ILIKE btrim(sqlc.narg('pacing_mode')::text)
  )
ORDER BY
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'clicks' AND COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN COALESCE(stats.stat_clicks, 0) END) DESC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'clicks' AND NOT COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN COALESCE(stats.stat_clicks, 0) END) ASC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'impressions' AND COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN COALESCE(stats.stat_impressions, 0) END) DESC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'impressions' AND NOT COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN COALESCE(stats.stat_impressions, 0) END) ASC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'conversions' AND COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN COALESCE(stats.stat_conversions, 0) END) DESC NULLS LAST,
  (CASE WHEN COALESCE(sqlc.narg('sort_field')::text, 'updated_at') = 'conversions' AND NOT COALESCE(sqlc.narg('sort_desc')::boolean, true) THEN COALESCE(stats.stat_conversions, 0) END) ASC NULLS LAST,
  updated_at DESC
LIMIT $1 OFFSET $2;

-- name: SumCampaignStatsByCampaignIDsInRange :many
SELECT
    campaign_id,
    COALESCE(SUM(impressions_count), 0)::bigint AS impressions,
    COALESCE(SUM(clicks_count), 0)::bigint AS clicks,
    COALESCE(SUM(conversions_count), 0)::bigint AS conversions
FROM campaign_stats
WHERE campaign_id = ANY(@campaign_ids::uuid[])
  AND date >= @from_date::date
  AND date <= @to_date::date
GROUP BY campaign_id;

-- name: ListCampaignIDsByCustomers :many
SELECT customer_id, id AS campaign_id
FROM campaigns
WHERE customer_id = ANY($1::uuid[])
  AND deleted_at IS NULL;

-- name: CountStatusHistory :one
SELECT COUNT(*) FROM campaign_status_history
WHERE campaign_id = $1;

-- name: ListStatusHistory :many
SELECT * FROM campaign_status_history
WHERE campaign_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreateBlacklistIP :one
INSERT INTO ip_blacklist (ip, reason, expires_at)
VALUES ($1, $2, $3)
ON CONFLICT (ip) DO UPDATE
    SET reason = EXCLUDED.reason,
        created_at = CURRENT_TIMESTAMP,
        expires_at = EXCLUDED.expires_at
RETURNING *;

-- name: CreateEdgeBlockAudit :one
INSERT INTO edge_block_audit (ip, reason_id, ttl, source)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: DeleteBlacklistIP :exec
DELETE FROM ip_blacklist
WHERE ip = $1;

-- name: ListExpiredBlacklistIPs :many
SELECT ip, reason FROM ip_blacklist
WHERE expires_at IS NOT NULL AND expires_at <= NOW()
ORDER BY expires_at ASC
LIMIT $1;

-- name: CountBlacklist :one
SELECT COUNT(*) FROM ip_blacklist;

-- name: ListBlacklist :many
SELECT * FROM ip_blacklist
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetAllBlacklist :many
SELECT ip, reason FROM ip_blacklist;

-- name: SetSystemSetting :exec
INSERT INTO system_settings (key, value, updated_at)
VALUES ($1, $2, CURRENT_TIMESTAMP)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = CURRENT_TIMESTAMP;

-- name: GetAllSystemSettings :many
SELECT key, value FROM system_settings;

-- name: GetSystemSetting :one
SELECT value FROM system_settings WHERE key = $1;

-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (event_type, payload)
VALUES ($1, $2)
RETURNING *;

-- name: CreateOutboxEventsBatch :exec
INSERT INTO outbox_events (event_type, payload)
SELECT unnest(@event_types::text[]), unnest(@payloads::bytea[]);

-- name: GetPendingOutboxEventsForUpdate :many
SELECT * FROM outbox_events
WHERE status = 'PENDING'
ORDER BY
  CASE event_type
    WHEN 'UPDATE_BLACKLIST' THEN 0
    WHEN 'ML_BLACKLIST_ADD' THEN 0
    WHEN 'ML_SCORE_BOOST' THEN 0
    WHEN 'ML_GHOST_IVT' THEN 0
    WHEN 'ML_SILENT_REJECT' THEN 0
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

-- name: MarkOutboxEventProcessed :exec
UPDATE outbox_events
SET status = 'PROCESSED'
WHERE id = $1;

-- name: GetDrainingCampaignsForUpdate :many
SELECT * FROM campaigns
WHERE status = 'DRAINING' AND updated_at < $1
ORDER BY updated_at ASC
LIMIT $2
FOR UPDATE SKIP LOCKED;

-- name: ListCustomersForScoring :many
SELECT 
    c.id,
    COALESCE(FLOOR(EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - c.created_at)) / 86400), 0)::integer AS age_days,
    COALESCE(SUM(l.amount), 0)::bigint AS topup_sum_30d
FROM customers c
LEFT JOIN balance_ledger l ON l.customer_id = c.id 
    AND (l.type = 'TOPUP' OR l.type = 'PAYMENT_TOPUP') 
    AND l.created_at >= CURRENT_TIMESTAMP - INTERVAL '30 days'
GROUP BY c.id;

-- name: UpdateCustomerOverdraft :one
UPDATE customers
SET allowed_overdraft = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: UpdateCustomerCostCenter :one
UPDATE customers
SET cost_center = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: CreateBrand :one
INSERT INTO advertiser_brands (id, customer_id, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetBrand :one
SELECT * FROM advertiser_brands WHERE id = $1 LIMIT 1;

-- name: GetBrandForUpdate :one
SELECT * FROM advertiser_brands WHERE id = $1 LIMIT 1 FOR UPDATE;

-- name: ConfigureBrandFcap :exec
UPDATE advertiser_brands
SET freq_limit = $2,
    freq_window = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: ListBrandsByCustomer :many
SELECT * FROM advertiser_brands
WHERE customer_id = $1
ORDER BY created_at DESC;

-- name: GetCampaignsWithStats :many
SELECT 
    c.id, c.name, c.status, c.budget_limit, c.created_at, c.updated_at, c.customer_id, c.current_spend, c.deleted_at, c.pacing_mode, c.daily_budget, c.timezone, c.freq_limit, c.freq_window, c.target_countries, c.brand_id, c.brand_fcap_key,
    COALESCE(SUM(s.impressions_count), 0)::bigint AS total_impressions,
    COALESCE(SUM(s.clicks_count), 0)::bigint AS total_clicks,
    COALESCE(SUM(s.conversions_count), 0)::bigint AS total_conversions
FROM campaigns c
LEFT JOIN campaign_stats s ON c.id = s.campaign_id
WHERE c.customer_id = $1 AND c.status = 'ACTIVE'
GROUP BY c.id;

-- name: UpdateCampaignBudget :one
UPDATE campaigns
SET budget_limit = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: GetAllActiveCampaignsWithStats :many
SELECT 
    c.id, c.name, c.status, c.budget_limit, c.created_at, c.updated_at, c.customer_id, c.current_spend, c.deleted_at, c.pacing_mode, c.daily_budget, c.timezone, c.freq_limit, c.freq_window, c.target_countries, c.brand_id, c.brand_fcap_key, c.daypart_hours,
    COALESCE(SUM(s.impressions_count), 0)::bigint AS total_impressions,
    COALESCE(SUM(s.clicks_count), 0)::bigint AS total_clicks,
    COALESCE(SUM(s.conversions_count), 0)::bigint AS total_conversions
FROM campaigns c
LEFT JOIN campaign_stats s ON c.id = s.campaign_id
WHERE c.status = 'ACTIVE'
GROUP BY c.id;

-- name: GetCampaignForUpdate :one
SELECT * FROM campaigns
WHERE id = $1
FOR UPDATE;

-- name: UpdateCampaignPacing :one
UPDATE campaigns
SET pacing_mode = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: UpdateCampaignAdmin :one
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
    attestation_mode = $14,
    dmr_enabled = $15,
    click_delivery = $16,
    proxy_upstream_url = $17,
    proxy_rewrite_assets = $18,
    tls_fingerprint_block_enabled = $19,
    conn_type_policy = $20,
    link_signing_enabled = $21,
    link_signing_ttl_sec = $22,
    cidr_block_enabled = $23,
    proxy_vpn_block_enabled = $24,
    moderator_intel_enabled = $25,
    review_traffic_action = $26,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: CountCampaignEvents :one
SELECT COUNT(*)::bigint FROM events WHERE campaign_id = $1;

-- name: ListCampaignEvents :many
SELECT click_id, event_type, user_id, payload, ip_address, user_agent, created_at
FROM events
WHERE campaign_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateCampaignFraudConfig :one
UPDATE campaigns
SET fraud_threshold_pass = $2,
    fraud_threshold_suspect = $3,
    fraud_threshold_ivt = $4,
    fraud_threshold_block = $5,
    silent_reject_enabled = $6,
    behavior_flags = $7,
    canvas_retest_enabled = $8,
    cgnat_ip_policy_enabled = $9,
    accept_lang_geo_enabled = $10,
    json_serialization_enabled = $11,
    conversion_reject_rules = $12,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;


-- name: SumLedgerSpendByCampaignWindow :many
SELECT 
    campaign_id,
    COALESCE(SUM(CASE WHEN amount < 0 THEN -amount ELSE 0 END), 0)::bigint AS total_spent_micro
FROM balance_ledger
WHERE created_at >= $1 
  AND created_at < $2
  AND (type = 'FEE' OR type = 'RECONCILIATION_ADJUST' OR type = 'REFUND')
GROUP BY campaign_id;

-- name: SumLedgerSpendByCampaignWindowWithCustomer :many
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

-- name: CreateReconRun :one
INSERT INTO recon_runs (period_start, period_end, status)
VALUES ($1, $2, 'PENDING')
RETURNING *;

-- name: UpdateReconRun :exec
UPDATE recon_runs
SET status = $2,
    total_delta = $3,
    campaigns_checked = $4,
    discrepancies_found = $5,
    completed_at = NOW()
WHERE id = $1;

-- name: InsertReconDiscrepancy :exec
INSERT INTO recon_discrepancies (
    run_id, campaign_id, customer_id, expected_spend, actual_spend, delta, redis_adjusted
) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: MaxCustomerReconLagMicro :one
SELECT COALESCE(MAX(ABS(delta)), 0)::bigint AS max_lag_micro
FROM recon_discrepancies
WHERE customer_id = $1
  AND created_at >= CURRENT_TIMESTAMP - INTERVAL '24 hours';

-- name: ListCustomerReconLagMicro :many
SELECT customer_id, COALESCE(MAX(ABS(delta)), 0)::bigint AS max_lag_micro
FROM recon_discrepancies
WHERE created_at >= CURRENT_TIMESTAMP - INTERVAL '24 hours'
GROUP BY customer_id;

-- name: SumCampaignStatsInRange :one
SELECT
    COALESCE(SUM(impressions_count), 0)::bigint AS impressions,
    COALESCE(SUM(clicks_count), 0)::bigint AS clicks,
    COALESCE(SUM(conversions_count), 0)::bigint AS conversions
FROM campaign_stats
WHERE campaign_id = @campaign_id
  AND date >= @from_date::date
  AND date <= @to_date::date;

-- name: SumCustomerCampaignStatsInRange :many
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

-- name: SumCustomerMarginWindowInRange :many
SELECT
  bl.campaign_id,
  COALESCE(SUM(CASE WHEN bl.type = 'FEE' THEN -bl.amount ELSE 0 END), 0)::bigint AS advertiser_spend_micro,
  COALESCE(SUM(CASE WHEN bl.type = 'rtb_cost' THEN bl.amount ELSE 0 END), 0)::bigint AS rtb_cost_micro,
  COALESCE(SUM(CASE WHEN bl.type = 'operator_margin' THEN bl.amount ELSE 0 END), 0)::bigint AS operator_margin_micro,
  COALESCE(SUM(CASE WHEN bl.type = 'publisher_payout' THEN bl.amount ELSE 0 END), 0)::bigint AS publisher_payout_micro
FROM balance_ledger bl
INNER JOIN campaigns c ON c.id = bl.campaign_id AND c.deleted_at IS NULL
WHERE c.customer_id = @customer_id
  AND bl.created_at >= @window_start
  AND bl.created_at < @window_end
  AND bl.type IN ('FEE', 'rtb_cost', 'operator_margin', 'publisher_payout')
GROUP BY bl.campaign_id;

-- name: ListSellers :many
SELECT * FROM sellers ORDER BY seller_id;

-- name: GetSeller :one
SELECT * FROM sellers WHERE id = $1;

-- name: CreateSeller :one
INSERT INTO sellers (seller_id, domain, seller_type, name, is_confidential)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateSeller :one
UPDATE sellers
SET seller_id = $2,
    domain = $3,
    seller_type = $4,
    name = $5,
    is_confidential = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteSeller :exec
DELETE FROM sellers WHERE id = $1;

-- name: ListAdsTxtEntries :many
SELECT * FROM ads_txt_entries ORDER BY sort_order, id;

-- name: GetAdsTxtEntry :one
SELECT * FROM ads_txt_entries WHERE id = $1;

-- name: CreateAdsTxtEntry :one
INSERT INTO ads_txt_entries (domain, publisher_account_id, relationship, cert_authority_id, sort_order)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateAdsTxtEntry :one
UPDATE ads_txt_entries
SET domain = $2,
    publisher_account_id = $3,
    relationship = $4,
    cert_authority_id = $5,
    sort_order = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteAdsTxtEntry :exec
DELETE FROM ads_txt_entries WHERE id = $1;

-- name: UpdateCampaignSupplyChain :one
UPDATE campaigns
SET supply_chain_nodes = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListRtbDeals :many
SELECT * FROM rtb_deals ORDER BY deal_id;

-- name: GetRtbDeal :one
SELECT * FROM rtb_deals WHERE id = $1;

-- name: GetRtbDealByDealID :one
SELECT * FROM rtb_deals WHERE deal_id = $1;

-- name: CreateRtbDeal :one
INSERT INTO rtb_deals (deal_id, floor_micro, geo_mask, cat_mask, pacing, customer_id, seats)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateRtbDeal :one
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

-- name: DeleteRtbDeal :exec
DELETE FROM rtb_deals WHERE id = $1;

-- name: UpsertRtbFloorSuggestion :exec
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

-- name: ListRtbFloorSuggestions :many
SELECT placement_id, deal_id, current_floor_micro, suggested_floor_micro,
       win_rate, sample_n, floor_bucket_micro, computed_at
FROM rtb_floor_suggestions
ORDER BY placement_id;

-- name: ListRtbFloorSuggestionsByPlacementIDs :many
SELECT placement_id, deal_id, current_floor_micro, suggested_floor_micro,
       win_rate, sample_n, floor_bucket_micro, computed_at
FROM rtb_floor_suggestions
WHERE placement_id = ANY($1::text[])
ORDER BY placement_id;

-- name: UpsertCampaignShardAssignment :one
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

-- name: GetCampaignShardAssignment :one
SELECT * FROM campaign_shard_assignment
WHERE campaign_id = $1;

-- name: DeleteCampaignShardAssignment :exec
DELETE FROM campaign_shard_assignment
WHERE campaign_id = $1;

-- name: GetCTVGtaxSettlement :one
SELECT *
FROM ctv_gtax_settlements
WHERE settlement_id = $1;

-- name: InsertCTVGtaxSettlement :one
INSERT INTO ctv_gtax_settlements (
  settlement_id, customer_id, campaign_id, spend_micro, tax_micro, fee_ledger_id, tax_ledger_id
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListActiveExperimentCohorts :many
SELECT id, name, active, salt, variants, created_at, updated_at
FROM experiment_cohorts
WHERE active = TRUE
ORDER BY name;

-- name: UpsertExperimentCohort :one
INSERT INTO experiment_cohorts (id, name, active, salt, variants)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    active = EXCLUDED.active,
    salt = EXCLUDED.salt,
    variants = EXCLUDED.variants,
    updated_at = now()
RETURNING *;

-- name: GetExperimentCohort :one
SELECT id, name, active, salt, variants, created_at, updated_at
FROM experiment_cohorts
WHERE id = $1;

-- name: ListExistingAlertRuleWindows :many
SELECT rule_id
FROM alert_rule_events
WHERE rule_id = ANY($1::uuid[])
  AND window_start = $2;

-- name: UpdateCampaignIngressCostConfig :one
UPDATE campaigns
SET ingress_cost_config = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: CloneFlowFromSource :one
INSERT INTO flows (id, name, paths)
SELECT $1, f.name || $2, f.paths
FROM flows f
WHERE f.id = $3
RETURNING id;

-- name: InsertClonedCampaign :one
INSERT INTO campaigns (
    id,
    name,
    status,
    budget_limit,
    current_spend,
    customer_id,
    pacing_mode,
    daily_budget,
    timezone,
    freq_limit,
    freq_window,
    target_countries,
    brand_id,
    brand_fcap_key,
    start_at,
    end_at,
    daypart_hours,
    template_id,
    fraud_threshold_pass,
    fraud_threshold_suspect,
    fraud_threshold_ivt,
    fraud_threshold_block,
    silent_reject_enabled,
    behavior_flags,
    supply_chain_nodes,
    require_consent_purposes,
    reserve_micro,
    target_url,
    creative_payload,
    referrer_filter,
    retarget_segment_id,
    segment_ttl_hours,
    segment_include,
    segment_exclude,
    safe_page_url,
    safe_page_enabled,
    cidr_block_enabled,
    click_delivery,
    proxy_upstream_url,
    proxy_rewrite_assets,
    integration_schema_id,
    owner_user_id,
    proxy_vpn_block_enabled,
    domain_pool_id,
    flow_id,
    status_integration_schema_id,
    dmr_enabled,
    tls_fingerprint_block_enabled,
    conn_type_policy,
    link_signing_enabled,
    link_signing_ttl_sec,
    attestation_enabled,
    attestation_ttl_sec,
    attestation_mode,
    social_in_app_enabled,
    moderator_intel_enabled,
    review_traffic_action,
    ingress_cost_config,
    traffic_template_id,
    click_query_params
)
SELECT
    $1,
    $2,
    $3,
    src.budget_limit,
    0,
    src.customer_id,
    src.pacing_mode,
    src.daily_budget,
    src.timezone,
    src.freq_limit,
    src.freq_window,
    src.target_countries,
    src.brand_id,
    $4,
    src.start_at,
    src.end_at,
    src.daypart_hours,
    src.template_id,
    src.fraud_threshold_pass,
    src.fraud_threshold_suspect,
    src.fraud_threshold_ivt,
    src.fraud_threshold_block,
    src.silent_reject_enabled,
    src.behavior_flags,
    src.supply_chain_nodes,
    src.require_consent_purposes,
    src.reserve_micro,
    src.target_url,
    src.creative_payload,
    src.referrer_filter,
    src.retarget_segment_id,
    src.segment_ttl_hours,
    src.segment_include,
    src.segment_exclude,
    src.safe_page_url,
    src.safe_page_enabled,
    src.cidr_block_enabled,
    src.click_delivery,
    src.proxy_upstream_url,
    src.proxy_rewrite_assets,
    src.integration_schema_id,
    src.owner_user_id,
    src.proxy_vpn_block_enabled,
    src.domain_pool_id,
    $5,
    src.status_integration_schema_id,
    src.dmr_enabled,
    src.tls_fingerprint_block_enabled,
    src.conn_type_policy,
    src.link_signing_enabled,
    src.link_signing_ttl_sec,
    src.attestation_enabled,
    src.attestation_ttl_sec,
    src.attestation_mode,
    src.social_in_app_enabled,
    src.moderator_intel_enabled,
    src.review_traffic_action,
    src.ingress_cost_config,
    src.traffic_template_id,
    src.click_query_params
FROM campaigns src
WHERE src.id = $6
  AND src.deleted_at IS NULL
RETURNING *;

