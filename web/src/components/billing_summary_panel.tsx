import { useCallback, useEffect, useState } from 'react';
import { to } from '../lib/to.js';
import { formatAmountMicro } from '../helpers/money.js';
import { fetchBillingSummary } from '../helpers/billing_admin_api.js';
import type { BillingSummaryDTO } from '../types/billing.js';
import { ErrorBlock } from './error_block.js';

export function BillingSummaryPanel() {
  const [data, setData] = useState<BillingSummaryDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    const [summary, err] = await to(fetchBillingSummary());
    setLoading(false);
    if (err) {
      setError(err);
      setData(null);
      return;
    }
    setData(summary ?? {});
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  if (loading) {
    return (
      <div className="section-card" data-testid="billing-summary-panel">
        <h3 className="subsection-title">Fleet billing summary</h3>
        <p className="text-muted text-sm">Loading…</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="section-card" data-testid="billing-summary-panel">
        <h3 className="subsection-title">Fleet billing summary</h3>
        <ErrorBlock error={error} />
      </div>
    );
  }

  return (
    <div className="section-card" data-testid="billing-summary-panel">
      <h3 className="subsection-title">Fleet billing summary (MTD)</h3>
      <div className="grid-stats">
        <div className="metric-card">
          <div className="metric-card__label">Invoiced MTD</div>
          <div className="metric-card__value">
            {formatAmountMicro(data?.invoiced_mtd_micro ?? 0)}
          </div>
        </div>
        <div className="metric-card">
          <div className="metric-card__label">Invoices</div>
          <div className="metric-card__value">{String(data?.invoice_count_mtd ?? 0)}</div>
        </div>
        <div className="metric-card">
          <div className="metric-card__label">Undelivered notifications</div>
          <div className="metric-card__value">
            {String(data?.undelivered_invoice_notifications ?? 0)}
          </div>
        </div>
        <div className="metric-card">
          <div className="metric-card__label">Customers with spend</div>
          <div className="metric-card__value">
            {String(data?.customers_with_spend_in_month ?? 0)}
          </div>
        </div>
      </div>
    </div>
  );
}
