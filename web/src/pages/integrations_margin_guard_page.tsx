import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { isCustomerUuid } from '../helpers/customer_context.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { scanMarginBreaches, type MarginBreachRow } from '../helpers/margin_guard_api.js';
import { formatMicro } from '../helpers/money.js';
import { Button, ButtonLink } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { StatusBadge } from '../components/status_badge.js';

function TableSkeleton({ cols, rows = 4 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }, (_, i) => (
        <tr key={`sk-${i}`} className="data-table__row--skeleton" aria-hidden="true">
          {Array.from({ length: cols }, (__, j) => (
            <td key={`sk-${i}-${j}`}>
              <span className="skeleton-bar" />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

export function IntegrationsMarginGuardPage() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const canWrite = can(user?.permissions ?? [], 'campaigns:write');
  const sessionScoped = hasBoundCustomer(user?.role);
  const tenantCustomerId = boundCustomerId(user);

  const [customerId, setCustomerId] = useState(
    sessionScoped ? tenantCustomerId : (searchParams.get('customer_id')?.trim() ?? '')
  );
  const [rows, setRows] = useState<MarginBreachRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [scanned, setScanned] = useState(false);

  const scan = useCallback(async () => {
    if (!isCustomerUuid(customerId)) {
      setRows([]);
      setScanned(false);
      return;
    }
    setLoading(true);
    setError(null);
    const result = await scanMarginBreaches(customerId);
    setLoading(false);
    setScanned(true);
    if (result.error) {
      setError(result.error);
      setRows([]);
      return;
    }
    setRows(result.rows);
    if (!sessionScoped && isCustomerUuid(customerId)) {
      const next = new URLSearchParams(searchParams);
      next.set('customer_id', customerId);
      setSearchParams(next, { replace: true });
    }
  }, [customerId, sessionScoped, searchParams, setSearchParams]);

  useEffect(() => {
    if (isCustomerUuid(customerId)) void scan();
  }, []); 

  const emptyMsg = !isCustomerUuid(customerId)
    ? 'Enter customer UUID and scan.'
    : scanned
      ? 'No margin breaches in active campaigns.'
      : 'Click Scan to load breaches.';

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Margin scan failed" />;
  }

  return (
    <section className="stack" data-testid="margin-guard-portfolio">
      <div className="page-header">
        <h1 className="page-header__title">Margin Guard</h1>
        <p className="page-header__desc">
          Active campaigns where RTB cost exceeds the revenue threshold (1h window). Open a row to
          edit policy.
        </p>
      </div>

      {!sessionScoped ? (
        <label className="form-field" htmlFor="mg-portfolio-customer">
          Customer UUID
          <input
            id="mg-portfolio-customer"
            className="form-input form-input--sm font-mono"
            value={customerId}
            data-testid="margin-guard-customer"
            onChange={(e) => setCustomerId(e.target.value.trim())}
          />
        </label>
      ) : (
        <p className="text-muted text-sm">
          Customer: <span className="font-mono">{customerId || '-'}</span>
        </p>
      )}

      <div className="flex gap-2 items-center">
        <Button
          label={loading ? 'Scanning...' : 'Scan campaigns'}
          variant="secondary"
          size="sm"
          loading={loading}
          disabled={loading || !isCustomerUuid(customerId)}
          data-testid="margin-guard-scan"
          onClick={() => void scan()}
        />
        {rows.length > 0 ? (
          <span className="text-muted text-sm" data-testid="margin-guard-count">
            {rows.length} breach{rows.length === 1 ? '' : 'es'}
          </span>
        ) : null}
      </div>

      <div className="table-wrapper elevation-raised">
        <table className="data-table">
          <thead>
            <tr>
              <th>Campaign</th>
              <th>Status</th>
              <th>Spend (1h)</th>
              <th>RTB cost</th>
              <th>Threshold (bps)</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {loading ? <TableSkeleton cols={6} /> : null}
            {!loading && rows.length === 0 ? (
              <tr>
                <td colSpan={6}>{emptyMsg}</td>
              </tr>
            ) : null}
            {rows.map(({ campaign: c, margin: m }) => (
              <tr key={c.id}>
                <td>{c.name ?? c.id}</td>
                <td>
                  <StatusBadge status={c.status} />
                </td>
                <td className="font-mono">${formatMicro(m.advertiser_spend_micro ?? 0)}</td>
                <td className="font-mono">${formatMicro(m.rtb_cost_micro ?? 0)}</td>
                <td>{String(m.threshold_bps ?? '-')}</td>
                <td>
                  <ButtonLink
                    label={canWrite ? 'Edit policy' : 'View'}
                    href={`/campaigns/${c.id}?tab=margin`}
                    variant="secondary"
                    size="sm"
                    data-testid="margin-guard-edit-policy"
                    onClick={(e) => {
                      e.preventDefault();
                      navigate(`/campaigns/${c.id}?tab=margin`);
                    }}
                  />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
