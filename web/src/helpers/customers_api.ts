import { api } from './api_client.js';
import { to } from '../lib/to.js';

export type Customer = {
  id?: string;
  name?: string;
  balance?: string;
  currency?: string;
  cost_center?: string;
  active_campaigns?: number;
  total_spend?: string;
  created_at?: string;
  updated_at?: string;
};

export type CustomerListResponse = {
  items?: Customer[];
  total?: number;
};

export type CustomerSortField = 'name' | 'created_at';
export type CustomerSortOrder = 'asc' | 'desc';

export type CustomerListParams = {
  limit: number;
  offset: number;
  sort: CustomerSortField;
  order: CustomerSortOrder;
};

export type BalanceLedgerEntry = {
  id?: number;
  customer_id?: string;
  campaign_id?: string;
  amount?: string;
  type?: string;
  created_at?: string;
};

export type CustomerBalance = {
  customer_id?: string;
  balance?: string;
  currency?: string;
  ledger?: BalanceLedgerEntry[];
};

export type CustomerLedgerListResponse = {
  items?: BalanceLedgerEntry[];
  total?: number;
  limit?: number;
  offset?: number;
};

export type TaxProfile = {
  customer_id?: string;
  country_code?: string;
  tax_region?: string;
  tax_scheme?: string;
  tax_rate_bps?: number;
};

export type BillingStatement = {
  customer_id?: string;
  opening_balance_micro?: number;
  closing_balance_micro?: number;
  period?: { from?: string; to?: string };
  lines?: Array<{ description?: string; amount_micro?: number }>;
};

export type BillingForecast = {
  customer_id?: string;
  month?: string;
  ledger_mtd_micro?: number;
  projected_month_end_micro?: number;
  ch_unavailable?: boolean;
  low_confidence?: boolean;
};

export type Wallet = {
  customer_id?: string;
  balance_micro?: number;
  currency?: string;
  burn_days_estimate?: number;
  payment_provider_configured?: boolean;
  payment_provider?: string;
};

export type PaymentHistoryRow = {
  intent_id?: string;
  amount_micro?: number;
  currency?: string;
  status?: string;
  provider?: string;
  created_at?: string;
};

export type PaymentHistoryListResponse = {
  items?: PaymentHistoryRow[];
  total?: number;
  limit?: number;
  offset?: number;
};

export type CampaignRow = {
  id?: string;
  name?: string;
  status?: string;
  customer_id?: string;
};

export type CampaignListResponse = {
  items?: CampaignRow[];
  total?: number;
};

export type CreateApiKeyResponse = {
  raw_key?: string;
  id?: string;
  name?: string;
};

export type CustomerDetailTab =
  | 'overview'
  | 'balance'
  | 'ledger'
  | 'statement'
  | 'forecast'
  | 'wallet'
  | 'tax'
  | 'payments'
  | 'campaigns'
  | 'api_keys';

export function buildCustomersListUrl(params: CustomerListParams): string {
  const qs = new URLSearchParams({
    limit: String(params.limit),
    offset: String(params.offset),
    sort: params.sort,
    order: params.order,
  });
  return `/api/v1/customers?${qs.toString()}`;
}

export function customerPath(customerId: string, suffix = ''): string {
  return `/api/v1/customers/${encodeURIComponent(customerId)}${suffix}`;
}

export async function getCustomer(customerId: string, signal?: AbortSignal): Promise<Customer> {
  const [result, err] = await to(api<Customer>(customerPath(customerId), { signal }));
  if (err) throw err;
  if (!result?.data) throw new Error('empty customer response');
  return result.data;
}

export async function getCustomerBalance(customerId: string, signal?: AbortSignal): Promise<CustomerBalance> {
  const [result, err] = await to(api<CustomerBalance>(customerPath(customerId, '/balance'), { signal }));
  if (err) throw err;
  return result?.data ?? {};
}

export async function getCustomerLedger(
  customerId: string,
  limit: number,
  offset: number,
  signal?: AbortSignal
): Promise<CustomerLedgerListResponse> {
  const qs = new URLSearchParams({ limit: String(limit), offset: String(offset) });
  const [result, err] = await to(
    api<CustomerLedgerListResponse>(`${customerPath(customerId, '/ledger')}?${qs}`, { signal })
  );
  if (err) throw err;
  return result?.data ?? {};
}

export function balanceExportUrl(customerId: string): string {
  return customerPath(customerId, '/balance/export');
}

export async function getBillingStatement(
  customerId: string,
  month: string,
  signal?: AbortSignal
): Promise<BillingStatement> {
  const qs = new URLSearchParams({ month });
  const [result, err] = await to(
    api<BillingStatement>(`${customerPath(customerId, '/billing/statement')}?${qs}`, { signal })
  );
  if (err) throw err;
  return result?.data ?? {};
}

export async function getBillingForecast(
  customerId: string,
  signal?: AbortSignal
): Promise<BillingForecast> {
  const [result, err] = await to(
    api<BillingForecast>(customerPath(customerId, '/billing/forecast'), { signal })
  );
  if (err) throw err;
  return result?.data ?? {};
}

export async function getCustomerWallet(customerId: string, signal?: AbortSignal): Promise<Wallet> {
  const [result, err] = await to(api<Wallet>(customerPath(customerId, '/wallet'), { signal }));
  if (err) throw err;
  return result?.data ?? {};
}

export async function getTaxProfile(customerId: string, signal?: AbortSignal): Promise<TaxProfile> {
  const [result, err] = await to(api<TaxProfile>(customerPath(customerId, '/tax-profile'), { signal }));
  if (err) throw err;
  return result?.data ?? {};
}

export async function putTaxProfile(
  customerId: string,
  body: TaxProfile
): Promise<TaxProfile> {
  const [result, err] = await to(
    api<TaxProfile>(customerPath(customerId, '/tax-profile'), {
      method: 'PUT',
      body: JSON.stringify(body),
    })
  );
  if (err) throw err;
  if (!result?.data) throw new Error('empty tax profile response');
  return result.data;
}

export async function getCustomerPayments(
  customerId: string,
  limit: number,
  offset: number,
  signal?: AbortSignal
): Promise<PaymentHistoryListResponse> {
  const qs = new URLSearchParams({ limit: String(limit), offset: String(offset) });
  const [result, err] = await to(
    api<PaymentHistoryListResponse>(`${customerPath(customerId, '/payments')}?${qs}`, { signal })
  );
  if (err) throw err;
  return result?.data ?? {};
}

export async function listCustomerCampaigns(
  customerId: string,
  limit: number,
  offset: number,
  signal?: AbortSignal
): Promise<CampaignListResponse> {
  const qs = new URLSearchParams({
    customer_id: customerId,
    limit: String(limit),
    offset: String(offset),
  });
  const [result, err] = await to(api<CampaignListResponse>(`/api/v1/campaigns?${qs}`, { signal }));
  if (err) throw err;
  return result?.data ?? {};
}

export async function createSelfServeApiKey(name: string): Promise<CreateApiKeyResponse> {
  const [result, err] = await to(
    api<CreateApiKeyResponse>('/api/v1/selfserve/api-keys', {
      method: 'POST',
      body: JSON.stringify({ name }),
    })
  );
  if (err) throw err;
  return result?.data ?? {};
}

export async function listCustomers(
  params: CustomerListParams,
  signal?: AbortSignal
): Promise<CustomerListResponse> {
  const [result, err] = await to(
    api<CustomerListResponse>(buildCustomersListUrl(params), { signal })
  );
  if (err) throw err;
  if (!result?.data) throw new Error('empty customers list response');
  return result.data;
}

export const CUSTOMER_DETAIL_TABS: Array<{ id: CustomerDetailTab; label: string }> = [
  { id: 'overview', label: 'Overview' },
  { id: 'balance', label: 'Balance' },
  { id: 'ledger', label: 'Ledger' },
  { id: 'statement', label: 'Statement' },
  { id: 'forecast', label: 'Forecast' },
  { id: 'wallet', label: 'Wallet' },
  { id: 'tax', label: 'Tax profile' },
  { id: 'payments', label: 'Payments' },
  { id: 'campaigns', label: 'Campaigns' },
  { id: 'api_keys', label: 'API keys' },
];

export function parseCustomerDetailTab(raw: string | null): CustomerDetailTab {
  const found = CUSTOMER_DETAIL_TABS.find((tab) => tab.id === raw);
  return found ? found.id : 'overview';
}

export function currentStatementMonth(): string {
  const now = new Date();
  const month = String(now.getUTCMonth() + 1).padStart(2, '0');
  return `${now.getUTCFullYear()}-${month}`;
}
