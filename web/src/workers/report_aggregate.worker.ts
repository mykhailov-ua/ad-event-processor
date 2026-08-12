/**
 * Enrich report rows with roi_pct in place to avoid per-row object spreads.
 */

type ReportAggregateWorkerSelf = {
  onmessage: ((ev: MessageEvent<unknown>) => void) | null;
  postMessage: (message: unknown) => void;
};

type AggregateRow = {
  spend_micro?: unknown;
  revenue_micro?: unknown;
  roi_pct?: number | null;
};

const reportAggregateSelf = self as unknown as ReportAggregateWorkerSelf;

/**
 * Narrow unknown payload rows to mutable aggregate objects.
 */
function asAggregateRow(value: unknown): AggregateRow | null {
  if (value === null || typeof value !== 'object') return null;
  return value as AggregateRow;
}

reportAggregateSelf.onmessage = (e: MessageEvent<unknown>) => {
  const rows = e.data;
  if (!Array.isArray(rows)) {
    reportAggregateSelf.postMessage([]);
    return;
  }
  for (let i = 0; i < rows.length; i++) {
    const row = asAggregateRow(rows[i]);
    if (!row) continue;
    const spend = Number(row.spend_micro ?? 0);
    const revenue = Number(row.revenue_micro ?? 0);
    row.roi_pct = spend > 0 ? ((revenue - spend) / spend) * 100 : null;
  }
  reportAggregateSelf.postMessage(rows);
};

export {};
