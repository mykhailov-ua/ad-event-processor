WITH doomed AS (
    SELECT e.click_id, e.created_date
    FROM events e
    WHERE e.created_at < @cutoff
    LIMIT @batch_limit
)
DELETE FROM events e
USING doomed d
WHERE e.click_id = d.click_id AND e.created_date = d.created_date;
