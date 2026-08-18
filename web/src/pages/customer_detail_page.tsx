import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { to } from '../lib/to.js';
import { apiBlob } from '../helpers/api_blob.js';
import { apiConfirmed } from '../helpers/confirmed_api.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import * as auth from '../helpers/auth.js';
import * as storage from '../helpers/storage.js';
import { ApiError } from '../helpers/api_client.js';
import { can, isTenantUser } from '../helpers/permissions.js';
import { touchCustomerContext } from '../helpers/customer_context.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { formatAmountMicro, formatUsdDecimal } from '../helpers/money.js';
import type { CampaignDTO, CampaignListResponse } from '../types/campaign.js';
import type { CustomerDTO, TaxProfileDTO } from '../types/customer.js';
import type { WalletBalanceDTO } from '../types/billing.js';
import { createSortState, sortRows, toggleSort } from '../lib/table_sort.js';
import { CustomerApiKeysSection } from '../components/customer_api_keys_section.js';
import { BillingForecastWidget } from '../components/billing_forecast_widget.js';
import { BillingStatementPanel } from '../components/billing_statement_panel.js';
import { BillingPaymentHistoryPanel } from '../components/billing_payment_history_panel.js';
import { useResource } from '../helpers/use_resource.js';
import { Breadcrumbs } from '../components/breadcrumbs.js';
import { Button, ButtonLink } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { FormField } from '../components/form_field.js';
import { Icon } from '../components/icon.js';
import { StatusBadge } from '../components/status_badge.js';

function TableSkeleton({ cols, rows = 3 }: { cols: number; rows?: number }) {
  return (
    <>
      {Array.from({ length: rows }, (_, rowIndex) => (
        <tr key={`skel-${rowIndex}`} className="data-table__row--skeleton" aria-hidden="true">
          {Array.from({ length: cols }, (__, colIndex) => (
            <td key={`skel-${rowIndex}-${colIndex}`}>
              <span className="skeleton-bar" />
            </td>
          ))}
        </tr>
      ))}
    </>
  );
}

export function CustomerDetailPage() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const user = auth.getUser();
  const tenant = isTenantUser(user?.role);
  const canWriteCustomer = can(user?.permissions ?? [], 'customers:write');
  const canCreateApiKey = can(user?.permissions ?? [], 'campaigns:write');
  const canReadCustomer = can(user?.permissions ?? [], 'customers:read');

  const [taxSaving, setTaxSaving] = useState(false);
  const [balanceExportBusy, setBalanceExportBusy] = useState(false);
  const [sortState, setSortState] = useState(() => createSortState('name', 'asc'));
  const [taxForm, setTaxForm] = useState({
    country_code: '',
    tax_region: '',
    tax_scheme: 'standard',
    tax_rate_bps: '0',
  });

  useEffect(() => {
    if (id) touchCustomerContext(id);
  }, [id]);

  const {
    data: customer,
    loading: customerLoading,
    error: customerError,
  } = useResource<CustomerDTO>(id ? `/api/v1/customers/${id}` : null);

  const { data: campaignsData, loading: campaignsLoading } = useResource<CampaignListResponse>(
    id ? `/api/v1/campaigns?customer_id=${encodeURIComponent(id)}&limit=10&offset=0` : null
  );

  const { data: wallet, loading: walletLoading } = useResource<WalletBalanceDTO>(
    id ? `/api/v1/customers/${id}/wallet` : null
  );

  const {
    data: taxProfile,
    loading: taxLoading,
    error: taxError,
    reload: reloadTax,
  } = useResource<TaxProfileDTO>(id ? `/api/v1/customers/${id}/tax-profile` : null);

  const taxSyncedRef = useRef<string | null>(null);
  useEffect(() => {
    if (!taxProfile || taxSyncedRef.current === id) return;
    taxSyncedRef.current = id;
    setTaxForm({
      country_code: taxProfile.country_code ?? '',
      tax_region: taxProfile.tax_region ?? '',
      tax_scheme: taxProfile.tax_scheme ?? 'standard',
      tax_rate_bps: String(taxProfile.tax_rate_bps ?? 0),
    });
  }, [taxProfile, id]);

  const campaigns = useMemo(
    () =>
      sortRows(campaignsData?.items ?? [], sortState, {
        name: (c: CampaignDTO) => c.name ?? '',
        status: (c: CampaignDTO) => c.status ?? '',
        budget_limit: (c: CampaignDTO) => Number(c.budget_limit ?? 0),
      }),
    [campaignsData?.items, sortState]
  );

  const onCampaignSort = (key: string) => {
    setSortState((prev) => {
      const next = { ...prev };
      toggleSort(next, key);
      return next;
    });
  };

  if (tenant && user?.customer_id && id !== user.customer_id) {
    return (
      <ErrorBlock
        error={new ApiError(403, 'FORBIDDEN', 'tenant boundary')}
        fallbackTitle="Access denied"
      />
    );
  }

  const openBilling = () => {
    if (!id) return;
    storage.setLastCustomerId(id);
    navigate(`/billing?customer_id=${encodeURIComponent(id)}`);
  };

  const exportBalanceCsv = async () => {
    if (balanceExportBusy || !id) return;
    setBalanceExportBusy(true);
    const [blob, err] = await to(
      apiBlob(`/api/v1/customers/${encodeURIComponent(id)}/balance/export?format=csv`)
    );
    setBalanceExportBusy(false);
    if (err) {
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      return;
    }
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = `balance-${id.slice(0, 8)}.csv`;
    anchor.click();
    URL.revokeObjectURL(url);
  };

  const saveTaxProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canWriteCustomer || taxSaving || !id) return;
    const body = {
      country_code: taxForm.country_code.trim(),
      tax_region: taxForm.tax_region.trim(),
      tax_scheme: taxForm.tax_scheme.trim(),
      tax_rate_bps: Number.parseInt(taxForm.tax_rate_bps, 10),
    };
    setTaxSaving(true);
    const [, err] = await to(
      apiConfirmed(`/api/v1/customers/${id}/tax-profile`, {
        method: 'PUT',
        body: JSON.stringify(body),
      })
    );
    setTaxSaving(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      const view = mapServiceError(err);
      pushToastMessage({ title: view.title, message: view.message, code: view.code });
      return;
    }
    pushToastMessage({ title: 'Tax profile saved', message: id });
    reloadTax();
  };

  if (customerLoading && !customer) {
    return (
      <div className="grid-stats section-block">
        {['Name', 'Balance', 'Currency', 'Created'].map((label) => (
          <div key={label} className="metric-card metric-card--loading">
            <div className="metric-card__label">{label}</div>
            <div className="metric-card__value">…</div>
          </div>
        ))}
      </div>
    );
  }

  if (customerError) {
    return <ErrorBlock error={customerError} fallbackTitle="Failed to load customer" />;
  }

  if (!customer) return null;

  return (
    <>
      <div className="page-header">
        <Breadcrumbs
          items={[
            { label: 'Customers', href: '/customers' },
            { label: customer.name ?? 'Customer' },
          ]}
        />
        <div className="page-header__row">
          <div className="flex items-center gap-2">
            <Icon name="users" size={20} className="text-muted" />
            <h1 className="page-header__title">{customer.name}</h1>
          </div>
          {canReadCustomer ? (
            <Button
              label={balanceExportBusy ? 'Exporting…' : 'Export balance CSV'}
              variant="secondary"
              size="sm"
              data-testid="customer-balance-export"
              loading={balanceExportBusy}
              disabled={balanceExportBusy}
              onClick={() => void exportBalanceCsv()}
            />
          ) : null}
        </div>
      </div>

      <div className="grid-stats section-block">
        <div className="metric-card">
          <div className="metric-card__label">Balance</div>
          <div className="metric-card__value font-mono">{formatUsdDecimal(customer.balance)}</div>
        </div>
        <div className="metric-card">
          <div className="metric-card__label">Currency</div>
          <div className="metric-card__value">{customer.currency ?? 'USD'}</div>
        </div>
        <div className="metric-card">
          <div className="metric-card__label">Active campaigns</div>
          <div className="metric-card__value">{String(customer.active_campaigns ?? 0)}</div>
        </div>
        <div className="metric-card">
          <div className="metric-card__label">Total spend</div>
          <div className="metric-card__value font-mono">
            {formatUsdDecimal(customer.total_spend)}
          </div>
        </div>
      </div>

      <section className="section-block">
        <h2 className="subsection-title">Details</h2>
        <dl className="definition-list">
          <dt>ID</dt>
          <dd className="font-mono text-secondary">{customer.id}</dd>
          <dt>Created</dt>
          <dd>{customer.created_at ? new Date(customer.created_at).toLocaleString() : '—'}</dd>
          <dt>Updated</dt>
          <dd>{customer.updated_at ? new Date(customer.updated_at).toLocaleString() : '—'}</dd>
        </dl>
      </section>

      <section className="section-block" data-testid="customer-tax-profile">
        <h2 className="subsection-title">Tax profile</h2>
        {taxLoading ? <span className="text-muted">Loading…</span> : null}
        {taxError && !taxProfile ? (
          <p className="text-muted text-sm">Tax profile not available.</p>
        ) : null}
        {canWriteCustomer ? (
          <form className="stack max-w-lg" onSubmit={(e) => void saveTaxProfile(e)}>
            <FormField label="Country code" htmlFor="tax-country">
              <input
                id="tax-country"
                className="form-input form-input--sm"
                value={taxForm.country_code}
                required
                onChange={(e) => setTaxForm((f) => ({ ...f, country_code: e.target.value }))}
              />
            </FormField>
            <FormField label="Tax region" htmlFor="tax-region">
              <input
                id="tax-region"
                className="form-input form-input--sm"
                value={taxForm.tax_region}
                onChange={(e) => setTaxForm((f) => ({ ...f, tax_region: e.target.value }))}
              />
            </FormField>
            <FormField label="Scheme" htmlFor="tax-scheme">
              <input
                id="tax-scheme"
                className="form-input form-input--sm"
                value={taxForm.tax_scheme}
                required
                onChange={(e) => setTaxForm((f) => ({ ...f, tax_scheme: e.target.value }))}
              />
            </FormField>
            <FormField label="Rate (bps)" htmlFor="tax-rate-bps">
              <input
                id="tax-rate-bps"
                className="form-input form-input--sm"
                type="number"
                min={0}
                value={taxForm.tax_rate_bps}
                onChange={(e) => setTaxForm((f) => ({ ...f, tax_rate_bps: e.target.value }))}
              />
            </FormField>
            <Button
              label={taxSaving ? 'Saving…' : 'Save tax profile'}
              variant="primary"
              size="sm"
              type="submit"
              loading={taxSaving}
              disabled={taxSaving}
            />
          </form>
        ) : taxProfile ? (
          <dl className="definition-list">
            <dt>Country</dt>
            <dd>{taxProfile.country_code ?? '—'}</dd>
            <dt>Region</dt>
            <dd>{taxProfile.tax_region ?? '—'}</dd>
            <dt>Scheme</dt>
            <dd>{taxProfile.tax_scheme ?? '—'}</dd>
            <dt>Rate (bps)</dt>
            <dd className="font-mono">{String(taxProfile.tax_rate_bps ?? 0)}</dd>
          </dl>
        ) : !taxLoading && !taxError ? (
          <p className="text-muted text-sm">No tax profile on file.</p>
        ) : null}
      </section>

      <CustomerApiKeysSection canCreate={canCreateApiKey} />

      <section className="section-block">
        <div className="flex items-center gap-2 mb-3">
          <h2 className="subsection-title">Wallet</h2>
          <Button label="Billing" variant="secondary" size="sm" onClick={openBilling} />
        </div>
        {walletLoading ? <span className="text-muted">Loading…</span> : null}
        {wallet ? (
          <div className="metric-row section-block">
            <div className="metric-card">
              <div className="metric-card__label">Balance (micro)</div>
              <div className="metric-card__value font-mono">
                {formatAmountMicro(wallet.balance_micro ?? 0, wallet.currency)}
              </div>
            </div>
            <div className="metric-card">
              <div className="metric-card__label">Overdraft</div>
              <div className="metric-card__value font-mono">
                {formatAmountMicro(wallet.allowed_overdraft_micro ?? 0, wallet.currency)}
              </div>
            </div>
          </div>
        ) : null}
      </section>

      <BillingForecastWidget customerId={id} />

      {can(user?.permissions ?? [], 'customers:read') ? (
        <div className="stack section-block">
          <BillingStatementPanel customerId={id} />
          <BillingPaymentHistoryPanel customerId={id} />
        </div>
      ) : null}

      <section className="section-block">
        <div className="flex items-center gap-2 mb-3">
          <h2 className="subsection-title">Campaigns</h2>
          <ButtonLink
            label="All campaigns"
            href={`/campaigns?customer_id=${encodeURIComponent(id)}`}
            variant="secondary"
            size="sm"
          />
        </div>
        {campaignsLoading && campaigns.length === 0 ? (
          <div className="table-wrapper elevation-raised">
            <table className="data-table">
              <tbody>
                <TableSkeleton cols={3} rows={3} />
              </tbody>
            </table>
          </div>
        ) : null}
        {!campaignsLoading && campaigns.length === 0 ? (
          <div className="empty-state">
            <div className="empty-state__title">No campaigns</div>
            <div className="empty-state__desc text-muted text-sm">
              This customer has no campaigns yet.
            </div>
            <Button
              label="View all campaigns"
              variant="secondary"
              size="sm"
              className="mt-3"
              onClick={() => navigate(`/campaigns?customer_id=${encodeURIComponent(id)}`)}
            />
          </div>
        ) : null}
        {!campaignsLoading && campaigns.length > 0 ? (
          <div className="table-wrapper elevation-raised">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">
                    <button
                      type="button"
                      className="data-table__sort"
                      onClick={() => onCampaignSort('name')}
                    >
                      Name
                    </button>
                  </th>
                  <th scope="col">
                    <button
                      type="button"
                      className="data-table__sort"
                      onClick={() => onCampaignSort('status')}
                    >
                      Status
                    </button>
                  </th>
                  <th scope="col">
                    <button
                      type="button"
                      className="data-table__sort"
                      onClick={() => onCampaignSort('budget_limit')}
                    >
                      Budget
                    </button>
                  </th>
                </tr>
              </thead>
              <tbody>
                {campaigns.map((c) => (
                  <tr
                    key={c.id}
                    className="data-table__row--clickable"
                    tabIndex={0}
                    role="link"
                    onClick={() => navigate(`/campaigns/${c.id}`)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault();
                        navigate(`/campaigns/${c.id}`);
                      }
                    }}
                  >
                    <td className="font-medium">{c.name}</td>
                    <td>
                      <StatusBadge status={c.status} />
                    </td>
                    <td className="font-mono">{formatUsdDecimal(c.budget_limit ?? '0.00')}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </section>
    </>
  );
}
