-- name: UpsertClickConversionLedger :one
INSERT INTO click_conversion_ledger (campaign_id, click_id, payout_micro, last_status, updated_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (campaign_id, click_id) DO UPDATE SET
    payout_micro = click_conversion_ledger.payout_micro + EXCLUDED.payout_micro,
    last_status = EXCLUDED.last_status,
    updated_at = now()
RETURNING campaign_id, click_id, payout_micro, last_status, updated_at;

-- name: GetClickConversionLedger :one
SELECT campaign_id, click_id, payout_micro, last_status, updated_at
FROM click_conversion_ledger
WHERE campaign_id = $1 AND click_id = $2;
