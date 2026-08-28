import { useState } from 'react';
import { isoDaysAgo, toIsoNow } from '../helpers/date_presets.js';
import { apiBlobResult } from '../helpers/api_blob.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { Button } from './button.js';

export type BillingUsageExportSectionProps = {
  customerId: string;
  costCenter?: string;
  tenant: boolean;
};

/**
 * Download operational usage meters (events/RPS aggregates) as CSV for pass-through billing.
 */
export function BillingUsageExportSection({
  customerId,
  costCenter,
  tenant,
}: BillingUsageExportSectionProps) {
  const [fromDate, setFromDate] = useState(() => isoDaysAgo(30).slice(0, 10));
  const [toDate, setToDate] = useState(() => toIsoNow().slice(0, 10));
  const [filterCostCenter, setFilterCostCenter] = useState(costCenter ?? '');
  const [downloading, setDownloading] = useState(false);

  const downloadUsage = async () => {
    if (downloading) return;
    setDownloading(true);
    const params = new URLSearchParams({
      format: 'csv',
      from: fromDate,
      to: toDate,
    });
    if (customerId) params.set('customer_id', customerId);
    if (!tenant && filterCostCenter.trim()) {
      params.set('cost_center', filterCostCenter.trim());
    }
    const [, err] = await to(
      apiBlobResult(`/api/v1/billing/usage/export?${params.toString()}`).then((result) => {
        const url = URL.createObjectURL(result.blob);
        const anchor = document.createElement('a');
        anchor.href = url;
        anchor.download = `usage-${customerId || 'all'}.csv`;
        anchor.click();
        URL.revokeObjectURL(url);
        if (result.truncated) {
          pushToastMessage({
            title: 'Export truncated',
            message: 'Download the next page with the cursor from response headers if needed.',
          });
        }
      })
    );
    setDownloading(false);
    if (err) {
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
    }
  };

  return (
    <section className="section-card stack" data-testid="billing-usage-export">
      <h2 className="subsection-title">Usage meter export</h2>
      <p className="text-muted text-sm">
        Operational usage from billing.usage_daily (events, accepted_events). Not a financial
        invoice; ledger remains the source of truth.
      </p>
      <div className="form-row">
        <label className="form-label">
          From
          <input
            className="input"
            type="date"
            value={fromDate}
            onChange={(e) => setFromDate(e.target.value)}
          />
        </label>
        <label className="form-label">
          To
          <input
            className="input"
            type="date"
            value={toDate}
            onChange={(e) => setToDate(e.target.value)}
          />
        </label>
        {!tenant ? (
          <label className="form-label">
            Cost center
            <input
              className="input"
              type="text"
              value={filterCostCenter}
              placeholder="optional agency filter"
              onChange={(e) => setFilterCostCenter(e.target.value)}
            />
          </label>
        ) : null}
      </div>
      <div>
        <Button
          label={downloading ? 'Downloading...' : 'Download usage CSV'}
          variant="secondary"
          size="sm"
          disabled={downloading || !fromDate || !toDate}
          onClick={downloadUsage}
        />
      </div>
    </section>
  );
}
