import { useState } from 'react';
import { to } from '../lib/to.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { fetchCustomerBillingStatement } from '../helpers/billing_admin_api.js';
import type { BillingStatementDTO } from '../types/api/billing.js';
import { formatAmountMicro } from '../helpers/money.js';
import { Button } from './button.js';
import { StatusBadge } from './status_badge.js';

export type BillingStatementPanelProps = {
  customerId: string;
};

/**
 * Admin billing statement (GET /customers/{id}/billing/statement).
 */
export function BillingStatementPanel({ customerId }: BillingStatementPanelProps) {
  const [month, setMonth] = useState(() => new Date().toISOString().slice(0, 7));
  const [loading, setLoading] = useState(false);
  const [statement, setStatement] = useState<BillingStatementDTO | null>(null);

  const load = async () => {
    if (!customerId || loading) return;
    setLoading(true);
    const [data, err] = await to(fetchCustomerBillingStatement(customerId, month));
    setLoading(false);
    if (err) {
      pushToastMessage({ title: 'Statement failed', message: mapServiceError(err).message });
      setStatement(null);
      return;
    }
    setStatement(data ?? null);
  };

  return (
    <section className="section-card stack" data-testid="billing-statement-panel">
      <h3 className="subsection-title">Billing statement</h3>
      <label className="form-field" htmlFor="admin-stmt-month">
        Month (YYYY-MM)
        <input
          id="admin-stmt-month"
          className="form-input form-input--sm"
          value={month}
          data-testid="billing-statement-month"
          onChange={(e) => setMonth(e.target.value)}
        />
      </label>
      <Button
        label={loading ? 'Loading…' : 'Load statement'}
        variant="secondary"
        size="sm"
        loading={loading}
        disabled={loading}
        data-testid="billing-statement-load"
        onClick={() => void load()}
      />
      {statement ? (
        <div className="stack mt-2" data-testid="billing-statement-result">
          <div className="grid-stats">
            <div className="metric-card">
              <div className="metric-card__label">Opening</div>
              <div className="metric-card__value font-mono">
                {formatAmountMicro(statement.opening_balance_micro ?? 0, statement.currency)}
              </div>
            </div>
            <div className="metric-card">
              <div className="metric-card__label">Closing</div>
              <div className="metric-card__value font-mono">
                {formatAmountMicro(statement.closing_balance_micro ?? 0, statement.currency)}
              </div>
            </div>
            {statement.reconciliation?.delta_micro != null ? (
              <div className="metric-card">
                <div className="metric-card__label">Reconciliation Δ</div>
                <div className="metric-card__value font-mono">
                  {formatAmountMicro(statement.reconciliation.delta_micro, statement.currency)}
                </div>
              </div>
            ) : null}
          </div>
          <dl className="definition-list">
            <dt>Period</dt>
            <dd>{`${statement.period?.from ?? '—'} → ${statement.period?.to ?? '—'}`}</dd>
            {statement.tax_breakdown?.scheme ? (
              <>
                <dt>Tax</dt>
                <dd>
                  {statement.tax_breakdown.scheme}
                  {' '}
                  ({statement.tax_breakdown.rate_bps ?? 0} bps)
                  {' · '}
                  {formatAmountMicro(statement.tax_breakdown.tax_micro ?? 0, statement.currency)}
                </dd>
              </>
            ) : null}
          </dl>
          {statement.invoices && statement.invoices.length > 0 ? (
            <div className="table-wrapper">
              <table className="data-table" aria-label="Statement invoices">
                <thead>
                  <tr>
                    <th>Month</th>
                    <th>Status</th>
                    <th>Total</th>
                  </tr>
                </thead>
                <tbody>
                  {statement.invoices.map((inv) => (
                    <tr key={inv.id ?? inv.billing_month}>
                      <td>{inv.billing_month ?? '—'}</td>
                      <td>{inv.status ? <StatusBadge status={inv.status} kind="invoice" /> : '—'}</td>
                      <td className="font-mono">
                        {formatAmountMicro(inv.total_micro ?? 0, inv.currency ?? statement.currency)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}
