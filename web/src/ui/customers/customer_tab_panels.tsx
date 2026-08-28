import { useState } from 'react';
import { Link } from 'react-router-dom';
import {
  balanceExportUrl,
  createSelfServeApiKey,
  currentStatementMonth,
  type Customer,
  type TaxProfile,
} from '../../helpers/customers_api.js';
import { apiConfirmed, ConfirmCancelledError } from '../../helpers/confirmed_api.js';
import { formatLocaleDateTime as formatDate } from '../../helpers/format_display.js';
import { formatAmountMicro, formatUsdDecimal } from '../../helpers/money.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import { useResource } from '../../helpers/use_resource.js';
import { mapServiceError } from '../../helpers/service_error.js';
import { Button } from '../system/button.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { PaginationBar } from '../system/pagination_bar.js';
import { StubBanner } from '../system/stub_banner.js';
import styles from './customer_detail.module.css';

const LEDGER_PAGE = 25;
const PAYMENTS_PAGE = 25;
const CAMPAIGNS_PAGE = 20;

function shortId(id: string | undefined): string {
  if (!id) return '-';
  return id.length > 12 ? `${id.slice(0, 8)}...` : id;
}

export function CustomerKpiStrip({ customer }: { customer: Customer }) {
  return (
    <div className={styles.kpiStrip}>
      <div className={styles.kpiCard}>
        <span className={styles.kpiLabel}>Balance</span>
        <span className={styles.kpiValue}>{formatUsdDecimal(customer.balance)}</span>
      </div>
      <div className={styles.kpiCard}>
        <span className={styles.kpiLabel}>Currency</span>
        <span className={styles.kpiValue}>{customer.currency ?? 'USD'}</span>
      </div>
      <div className={styles.kpiCard}>
        <span className={styles.kpiLabel}>Active campaigns</span>
        <span className={styles.kpiValue}>{String(customer.active_campaigns ?? 0)}</span>
      </div>
      <div className={styles.kpiCard}>
        <span className={styles.kpiLabel}>Total spend</span>
        <span className={styles.kpiValue}>{formatUsdDecimal(customer.total_spend)}</span>
      </div>
    </div>
  );
}

export function CustomerToolbar({ customerId }: { customerId: string }) {
  return (
    <div className={styles.toolbar}>
      <Button
        variant="secondary"
       
        type="button"
        onClick={() => {
          window.location.assign(balanceExportUrl(customerId));
        }}
      >
        Export balance CSV
      </Button>
      <Link to="/billing">
        <Button variant="secondary">
          Open billing
        </Button>
      </Link>
    </div>
  );
}

export function CustomerOverviewPanel({ customer }: { customer: Customer }) {
  return (
    <dl className={styles.dl}>
      <dt>ID</dt>
      <dd className={styles.mono}>{customer.id ?? '-'}</dd>
      <dt>Name</dt>
      <dd>{customer.name ?? '-'}</dd>
      <dt>Cost center</dt>
      <dd>{customer.cost_center ?? '-'}</dd>
      <dt>Created</dt>
      <dd>{formatDate(customer.created_at)}</dd>
      <dt>Updated</dt>
      <dd>{formatDate(customer.updated_at)}</dd>
    </dl>
  );
}

export function CustomerBalancePanel({ customerId }: { customerId: string }) {
  const { data, loading, error } = useResource(
    `/api/v1/customers/${encodeURIComponent(customerId)}/balance`
  );

  if (loading) return <PageSkeleton rows={4} />;
  if (error) return <ErrorBlock error={error} fallbackTitle="Failed to load balance" />;

  const balance = data as {
    balance?: string;
    currency?: string;
    ledger?: Array<{ type?: string; amount?: string; created_at?: string }>;
  };

  return (
    <div className={styles.panel}>
      <dl className={styles.dl}>
        <dt>Balance</dt>
        <dd>{formatUsdDecimal(balance.balance)}</dd>
        <dt>Currency</dt>
        <dd>{balance.currency ?? 'USD'}</dd>
      </dl>
      <div className={styles.table}>
        <div className={styles.tableHead}>
          <span>Type</span>
          <span>Amount</span>
          <span>Created</span>
          <span />
        </div>
        {(balance.ledger ?? []).map((row, index) => (
          <div key={`${row.created_at ?? index}`} className={styles.tableRow}>
            <span>{row.type ?? '-'}</span>
            <span>{formatUsdDecimal(row.amount)}</span>
            <span>{formatDate(row.created_at)}</span>
            <span />
          </div>
        ))}
      </div>
    </div>
  );
}

export function CustomerLedgerPanel({ customerId }: { customerId: string }) {
  const [offset, setOffset] = useState(0);
  const url = `/api/v1/customers/${encodeURIComponent(customerId)}/ledger?limit=${LEDGER_PAGE}&offset=${offset}`;
  const { data, loading, error } = useResource(url);

  if (error) return <ErrorBlock error={error} fallbackTitle="Failed to load ledger" />;

  const ledger = data as {
    items?: Array<{ type?: string; amount?: string; created_at?: string; campaign_id?: string }>;
    total?: number;
  };

  return (
    <div className={styles.panel}>
      {loading && !ledger.items?.length ? <PageSkeleton rows={5} /> : null}
      <div className={styles.table}>
        <div className={styles.tableHead}>
          <span>Type</span>
          <span>Amount</span>
          <span>Campaign</span>
          <span>Created</span>
        </div>
        {(ledger.items ?? []).map((row, index) => (
          <div key={`${row.created_at ?? index}`} className={styles.tableRow}>
            <span>{row.type ?? '-'}</span>
            <span>{formatUsdDecimal(row.amount)}</span>
            <span className={styles.mono}>{shortId(row.campaign_id)}</span>
            <span>{formatDate(row.created_at)}</span>
          </div>
        ))}
      </div>
      <PaginationBar
        limit={LEDGER_PAGE}
        offset={offset}
        total={ledger.total ?? 0}
        onOffsetChange={setOffset}
      />
    </div>
  );
}

export function CustomerStatementPanel({ customerId }: { customerId: string }) {
  const month = currentStatementMonth();
  const url = `/api/v1/customers/${encodeURIComponent(customerId)}/billing/statement?month=${month}`;
  const { data, loading, error } = useResource(url);

  if (loading) return <PageSkeleton rows={4} />;
  if (error) return <ErrorBlock error={error} fallbackTitle="Failed to load statement" />;

  const statement = data as {
    opening_balance_micro?: number;
    closing_balance_micro?: number;
    period?: { from?: string; to?: string };
    lines?: Array<{ description?: string; amount_micro?: number }>;
  };

  return (
    <div className={styles.panel}>
      <dl className={styles.dl}>
        <dt>Month</dt>
        <dd>{month}</dd>
        <dt>Opening</dt>
        <dd>{formatAmountMicro(statement.opening_balance_micro)}</dd>
        <dt>Closing</dt>
        <dd>{formatAmountMicro(statement.closing_balance_micro)}</dd>
      </dl>
      <div className={styles.table}>
        <div className={styles.tableHead}>
          <span>Description</span>
          <span>Amount</span>
          <span />
          <span />
        </div>
        {(statement.lines ?? []).map((line, index) => (
          <div key={`${line.description ?? index}`} className={styles.tableRow}>
            <span>{line.description ?? '-'}</span>
            <span>{formatAmountMicro(line.amount_micro)}</span>
            <span />
            <span />
          </div>
        ))}
      </div>
    </div>
  );
}

export function CustomerForecastPanel({ customerId }: { customerId: string }) {
  const url = `/api/v1/customers/${encodeURIComponent(customerId)}/billing/forecast`;
  const { data, loading, error } = useResource(url);

  if (loading) return <PageSkeleton rows={3} />;
  if (error) {
    const view = mapServiceError(error);
    if (view.status === 503 || view.code === 'CLICKHOUSE_UNAVAILABLE') {
      return (
        <StubBanner
          title="Forecast unavailable"
          message="ClickHouse forecast data is not available for this customer."
        />
      );
    }
    return <ErrorBlock error={error} fallbackTitle="Failed to load forecast" />;
  }

  const forecast = data as {
    month?: string;
    ledger_mtd_micro?: number;
    projected_month_end_micro?: number;
    ch_unavailable?: boolean;
    low_confidence?: boolean;
  };

  if (forecast.ch_unavailable) {
    return (
      <StubBanner
        title="Forecast unavailable"
        message="ClickHouse data is unavailable. Ledger run-rate may still be shown when present."
      />
    );
  }

  return (
    <dl className={styles.dl}>
      <dt>Month</dt>
      <dd>{forecast.month ?? '-'}</dd>
      <dt>Ledger MTD</dt>
      <dd>{formatAmountMicro(forecast.ledger_mtd_micro)}</dd>
      <dt>Projected month end</dt>
      <dd>{formatAmountMicro(forecast.projected_month_end_micro)}</dd>
      <dt>Low confidence</dt>
      <dd>{forecast.low_confidence ? 'yes' : 'no'}</dd>
    </dl>
  );
}

export function CustomerWalletPanel({ customerId }: { customerId: string }) {
  const url = `/api/v1/customers/${encodeURIComponent(customerId)}/wallet`;
  const { data, loading, error } = useResource(url);

  if (loading) return <PageSkeleton rows={4} />;
  if (error) return <ErrorBlock error={error} fallbackTitle="Failed to load wallet" />;

  const wallet = data as {
    balance_micro?: number;
    currency?: string;
    burn_days_estimate?: number;
    payment_provider_configured?: boolean;
    payment_provider?: string;
  };

  return (
    <dl className={styles.dl}>
      <dt>Balance</dt>
      <dd>{formatAmountMicro(wallet.balance_micro, wallet.currency)}</dd>
      <dt>Provider</dt>
      <dd>{wallet.payment_provider ?? '-'}</dd>
      <dt>Provider configured</dt>
      <dd>{wallet.payment_provider_configured ? 'yes' : 'no'}</dd>
      <dt>Burn days estimate</dt>
      <dd>{wallet.burn_days_estimate != null ? String(wallet.burn_days_estimate) : '-'}</dd>
    </dl>
  );
}

export function CustomerTaxPanel({ customerId }: { customerId: string }) {
  const url = `/api/v1/customers/${encodeURIComponent(customerId)}/tax-profile`;
  const { data, loading, error, reload } = useResource(url);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<unknown>(null);
  const [form, setForm] = useState<TaxProfile>({});

  const profile = data as TaxProfile | null;

  if (loading && !profile) return <PageSkeleton rows={4} />;
  if (error) return <ErrorBlock error={error} fallbackTitle="Failed to load tax profile" />;

  const values = {
    country_code: form.country_code ?? profile?.country_code ?? '',
    tax_region: form.tax_region ?? profile?.tax_region ?? '',
    tax_scheme: form.tax_scheme ?? profile?.tax_scheme ?? '',
    tax_rate_bps:
      form.tax_rate_bps != null
        ? String(form.tax_rate_bps)
        : profile?.tax_rate_bps != null
          ? String(profile.tax_rate_bps)
          : '',
  };

  const onSave = async () => {
    setSaving(true);
    setSaveError(null);
    const body: TaxProfile = {
      country_code: values.country_code,
      tax_region: values.tax_region,
      tax_scheme: values.tax_scheme,
      tax_rate_bps: Number.parseInt(values.tax_rate_bps, 10) || 0,
    };
    try {
      await apiConfirmed(url, { method: 'PUT', body: JSON.stringify(body) });
      pushToastMessage({ title: 'Tax profile saved', message: 'Tax profile updated' });
      reload();
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setSaveError(err);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className={styles.panel}>
      {saveError ? <ErrorBlock error={saveError} fallbackTitle="Save failed" /> : null}
      <form
        className={styles.form}
        onSubmit={(e) => {
          e.preventDefault();
          void onSave();
        }}
      >
        <label className={styles.field}>
          <span className={styles.label}>Country code</span>
          <input
            className={styles.input}
            value={values.country_code}
            onChange={(e) => setForm({ ...form, country_code: e.target.value })}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Tax region</span>
          <input
            className={styles.input}
            value={values.tax_region}
            onChange={(e) => setForm({ ...form, tax_region: e.target.value })}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Tax scheme</span>
          <input
            className={styles.input}
            value={values.tax_scheme}
            onChange={(e) => setForm({ ...form, tax_scheme: e.target.value })}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.label}>Tax rate (bps)</span>
          <input
            className={styles.input}
            inputMode="numeric"
            value={values.tax_rate_bps}
            onChange={(e) => setForm({ ...form, tax_rate_bps: Number.parseInt(e.target.value, 10) })}
          />
        </label>
        <div className={styles.actions}>
          <Button type="submit" variant="primary" disabled={saving}>
            {saving ? 'Saving...' : 'Save tax profile'}
          </Button>
        </div>
      </form>
    </div>
  );
}

export function CustomerPaymentsPanel({ customerId }: { customerId: string }) {
  const [offset, setOffset] = useState(0);
  const url = `/api/v1/customers/${encodeURIComponent(customerId)}/payments?limit=${PAYMENTS_PAGE}&offset=${offset}`;
  const { data, loading, error } = useResource(url);

  if (error) return <ErrorBlock error={error} fallbackTitle="Failed to load payments" />;

  const payments = data as {
    items?: Array<{
      status?: string;
      amount_micro?: number;
      currency?: string;
      provider?: string;
      created_at?: string;
    }>;
    total?: number;
  };

  return (
    <div className={styles.panel}>
      {loading && !payments.items?.length ? <PageSkeleton rows={4} /> : null}
      <div className={styles.table}>
        <div className={styles.tableHead}>
          <span>Status</span>
          <span>Amount</span>
          <span>Provider</span>
          <span>Created</span>
        </div>
        {(payments.items ?? []).map((row, index) => (
          <div key={`${row.created_at ?? index}`} className={styles.tableRow}>
            <span>{row.status ?? '-'}</span>
            <span>{formatAmountMicro(row.amount_micro, row.currency)}</span>
            <span>{row.provider ?? '-'}</span>
            <span>{formatDate(row.created_at)}</span>
          </div>
        ))}
      </div>
      <PaginationBar
        limit={PAYMENTS_PAGE}
        offset={offset}
        total={payments.total ?? 0}
        onOffsetChange={setOffset}
      />
    </div>
  );
}

export function CustomerCampaignsPanel({ customerId }: { customerId: string }) {
  const url = `/api/v1/campaigns?customer_id=${encodeURIComponent(customerId)}&limit=${CAMPAIGNS_PAGE}&offset=0`;
  const { data, loading, error } = useResource(url);

  if (loading) return <PageSkeleton rows={4} />;
  if (error) return <ErrorBlock error={error} fallbackTitle="Failed to load campaigns" />;

  const campaigns = data as {
    items?: Array<{ id?: string; name?: string; status?: string }>;
    total?: number;
  };

  return (
    <div className={styles.panel}>
      <p className={styles.mono}>
        <Link to={`/campaigns?customer_id=${encodeURIComponent(customerId)}`}>
          Open full campaigns directory
        </Link>
      </p>
      <div className={styles.table}>
        <div className={styles.tableHead}>
          <span>Name</span>
          <span>Status</span>
          <span>ID</span>
          <span />
        </div>
        {(campaigns.items ?? []).map((row) => (
          <div key={row.id} className={styles.tableRow}>
            <span>{row.name ?? '-'}</span>
            <span>{row.status ?? '-'}</span>
            <span className={styles.mono}>{shortId(row.id)}</span>
            <span />
          </div>
        ))}
      </div>
      <span className={styles.kpiLabel}>{`${campaigns.total ?? 0} total`}</span>
    </div>
  );
}

export function CustomerApiKeysPanel() {
  const [name, setName] = useState('');
  const [rawKey, setRawKey] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<unknown>(null);

  return (
    <div className={styles.panel}>
      <StubBanner
        title="List endpoint not in OpenAPI"
        message="Only POST /api/v1/selfserve/api-keys is documented. Existing keys cannot be listed from this tab."
      />
      {error ? <ErrorBlock error={error} fallbackTitle="API key create failed" /> : null}
      {rawKey ? (
        <p className={styles.mono}>
          New key (copy now):
          {rawKey}
        </p>
      ) : null}
      <form
        className={styles.form}
        onSubmit={(e) => {
          e.preventDefault();
          setCreating(true);
          setError(null);
          void createSelfServeApiKey(name.trim())
            .then((res) => {
              setRawKey(res.raw_key ?? null);
              pushToastMessage({
                title: 'API key created',
                message: res.raw_key ? 'Copy the key now; it is shown once.' : 'Key created',
              });
            })
            .catch((err) => setError(err))
            .finally(() => setCreating(false));
        }}
      >
        <label className={styles.field}>
          <span className={styles.label}>Key name</span>
          <input
            className={styles.input}
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </label>
        <Button type="submit" variant="primary" disabled={creating || !name.trim()}>
          {creating ? 'Creating...' : 'Create API key'}
        </Button>
      </form>
    </div>
  );
}
