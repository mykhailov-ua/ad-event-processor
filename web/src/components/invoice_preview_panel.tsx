import { useState } from 'react';
import { to } from '../lib/to.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { previewBillingInvoice } from '../helpers/billing_admin_api.js';
import type { InvoicePreviewDTO } from '../types/billing.js';
import { formatAmountMicro } from '../helpers/money.js';
import { Button } from './button.js';
import { AlertBanner } from './alert_banner.js';

export type InvoicePreviewPanelProps = {
  customerId: string;
};

export function InvoicePreviewPanel({ customerId }: InvoicePreviewPanelProps) {
  const [month, setMonth] = useState(() => new Date().toISOString().slice(0, 7));
  const [loading, setLoading] = useState(false);
  const [preview, setPreview] = useState<InvoicePreviewDTO | null>(null);

  const runPreview = async () => {
    if (!customerId || loading) return;
    setLoading(true);
    const [data, err] = await to(previewBillingInvoice(customerId, month));
    setLoading(false);
    if (err) {
      pushToastMessage({ title: 'Preview failed', message: mapServiceError(err).message });
      setPreview(null);
      return;
    }
    setPreview(data ?? null);
  };

  return (
    <section className="section-card stack" data-testid="invoice-preview-panel">
      <h3 className="subsection-title">Invoice preview</h3>
      <p className="text-muted text-sm">
        Dry-run totals for a billing month - does not create or send an invoice.
      </p>
      <label className="form-field" htmlFor="invoice-preview-month">
        Billing month
        <input
          id="invoice-preview-month"
          className="form-input form-input--sm"
          value={month}
          data-testid="invoice-preview-month"
          onChange={(e) => setMonth(e.target.value)}
        />
      </label>
      <Button
        label={loading ? 'Previewing...' : 'Preview invoice'}
        variant="secondary"
        size="sm"
        loading={loading}
        disabled={loading}
        data-testid="invoice-preview-submit"
        onClick={() => void runPreview()}
      />
      {preview?.would_skip ? (
        <AlertBanner
          variant="info"
          message="Preview indicates invoice would be skipped (zero spend or already issued)."
        />
      ) : null}
      {preview ? (
        <div className="stack mt-2" data-testid="invoice-preview-result">
          <div className="grid-stats">
            <div className="metric-card">
              <div className="metric-card__label">Subtotal</div>
              <div className="metric-card__value font-mono">
                {formatAmountMicro(preview.subtotal_micro ?? 0, preview.currency)}
              </div>
            </div>
            <div className="metric-card">
              <div className="metric-card__label">Tax</div>
              <div className="metric-card__value font-mono">
                {formatAmountMicro(preview.tax_micro ?? 0, preview.currency)}
              </div>
            </div>
            <div className="metric-card">
              <div className="metric-card__label">Total</div>
              <div className="metric-card__value font-mono" data-testid="invoice-preview-total">
                {formatAmountMicro(preview.total_micro ?? 0, preview.currency)}
              </div>
            </div>
          </div>
          {preview.lines && preview.lines.length > 0 ? (
            <div className="table-wrapper">
              <table className="data-table" aria-label="Preview lines">
                <thead>
                  <tr>
                    <th>Type</th>
                    <th>Amount</th>
                    <th>Entries</th>
                  </tr>
                </thead>
                <tbody>
                  {preview.lines.map((line) => (
                    <tr key={line.ledger_type ?? String(line.amount_micro)}>
                      <td>{line.ledger_type ?? '-'}</td>
                      <td className="font-mono">
                        {formatAmountMicro(line.amount_micro ?? 0, preview.currency)}
                      </td>
                      <td>{String(line.entry_count ?? 0)}</td>
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
