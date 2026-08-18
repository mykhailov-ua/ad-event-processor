\echo '=== CountInvoicesAdmin (runs first with PaginatedList) ==='
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT COUNT(*)::bigint
FROM billing.invoices
WHERE (NULL::uuid IS NULL OR customer_id = NULL::uuid)
  AND (NULL::date IS NULL OR billing_month = NULL::date)
  AND (''::text = '' OR status::text = ''::text)
  AND (0::bigint = 0 OR total_micro >= 0::bigint);

\echo ''
\echo '=== ListInvoicesAdmin (skipped when count is 0) ==='
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT *
FROM billing.invoices
WHERE (NULL::uuid IS NULL OR customer_id = NULL::uuid)
  AND (NULL::date IS NULL OR billing_month = NULL::date)
  AND (''::text = '' OR status::text = ''::text)
  AND (0::bigint = 0 OR total_micro >= 0::bigint)
ORDER BY billing_month DESC, created_at DESC
LIMIT 50 OFFSET 0;
