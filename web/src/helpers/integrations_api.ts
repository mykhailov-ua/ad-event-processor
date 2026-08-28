import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type CostSyncExtraField = {
  key?: string;
  label?: string;
  required?: boolean;
  secret?: boolean;
  placeholder?: string;
  hint?: string;
};

export type CostSyncNetworkSchema = {
  network?: string;
  label?: string;
  account_id_label?: string;
  extra_fields?: CostSyncExtraField[];
};

export type CostSyncTokenMapping = {
  campaign_id_field?: string;
  date_field?: string;
  cost_field?: string;
};

export type CostSyncCredential = {
  customer_id?: string;
  network?: string;
  account_id?: string;
  extra_config?: Record<string, string>;
  extra_config_set?: Record<string, boolean>;
  sync_interval_minutes?: number;
  token_mapping?: CostSyncTokenMapping;
  token_expires_at?: string;
  updated_at?: string;
};

export type UpsertCostSyncCredentialBody = {
  customer_id: string;
  account_id?: string;
  access_token?: string;
  refresh_token?: string;
  api_key?: string;
  extra_config?: Record<string, string>;
  sync_interval_minutes?: number;
  token_mapping?: CostSyncTokenMapping;
};

export type RunCostSyncBody = {
  customer_id?: string;
  network?: string;
  from?: string;
  to?: string;
};

export type CostSyncRun = {
  id?: number;
  customer_id?: string;
  network?: string;
  cost_date?: string;
  status?: string;
  rows_imported?: number;
  total_amount_usd_micro?: number;
  error_message?: string;
  trigger_source?: string;
  started_at?: string;
  completed_at?: string;
};

export type PostbackCampaignStatus = {
  campaign_id?: string;
  provider?: string;
  last_success_at?: string;
  dlq_pending_count?: number;
};

export type PostbackDlqRow = {
  id?: number;
  outbox_event_id?: number;
  campaign_id?: string;
  click_id?: string;
  event_type?: string;
  payload?: unknown;
  failures_count?: number;
  last_error?: string;
  status?: string;
};

export type IntegrationSchema = {
  id?: string;
  name?: string;
  version?: number;
  kind?: string;
  schema?: unknown;
  created_at?: string;
  updated_at?: string;
};

export type CreateIntegrationSchemaBody = {
  name: string;
  version: number;
  schema: unknown;
};

export type ApplyIntegrationSchemaBody = {
  campaign_id: string;
};

export type IntegrationTemplateCatalogEntry = {
  name?: string;
  file?: string;
  version?: number;
  category?: string;
  kind?: string;
};

export type ImportIntegrationTemplatesBody = {
  names?: string[];
};

export type SupplySeller = {
  id?: number;
  seller_id?: string;
  domain?: string;
  seller_type?: string;
  name?: string;
  is_confidential?: boolean;
  created_at?: string;
  updated_at?: string;
};

export type SupplySellerWriteBody = {
  seller_id: string;
  domain: string;
  seller_type: string;
  name: string;
  is_confidential: boolean;
};

export type SupplyAdsTxtEntry = {
  id?: number;
  domain?: string;
  publisher_account_id?: string;
  relationship?: string;
  cert_authority_id?: string;
  sort_order?: number;
  created_at?: string;
  updated_at?: string;
};

export type SupplyAdsTxtWriteBody = {
  domain: string;
  publisher_account_id: string;
  relationship: string;
  cert_authority_id?: string;
  sort_order?: number;
};

export type SupplyValidation = {
  sellers_json_valid?: boolean;
  sellers_checksum_sha256?: string;
  sellers_count?: number;
  ads_txt_valid?: boolean;
  ads_txt_checksum_sha256?: string;
  ads_txt_line_count?: number;
  issues?: string[];
};

export type SupplyExportPath = {
  path?: string;
};

export type MarginGuardPolicy = {
  id?: string;
  campaign_id?: string;
  name?: string;
  min_clicks?: number;
  roi_floor_pct?: number;
  zero_conv_streak?: number;
  cost_over_revenue_threshold_bps?: number;
  is_active?: boolean;
};

export type MarginGuardActivity = {
  id?: string;
  policy_id?: string;
  campaign_id?: string;
  placement_id?: string;
  action?: string;
  reason?: string;
  metrics?: unknown;
  created_at?: string;
};

export type MarginGuardOverrideBody = {
  campaign_id: string;
  placement_id: string;
};

export type SmartAlertRule = {
  id?: string;
  customer_id?: string;
  campaign_id?: string;
  name?: string;
  metric?: string;
  operator?: string;
  threshold?: number;
  window_minutes?: number;
  webhook_url?: string;
  enabled?: boolean;
  created_at?: string;
  updated_at?: string;
};

export type UpsertSmartAlertRuleBody = {
  customer_id: string;
  campaign_id?: string;
  name: string;
  metric: string;
  operator: string;
  threshold: number;
  window_minutes: number;
  webhook_url: string;
  enabled: boolean;
};

export type SmartAlertEvent = {
  id?: string;
  rule_id?: string;
  customer_id?: string;
  campaign_id?: string;
  window_start?: string;
  window_end?: string;
  metric?: string;
  operator?: string;
  threshold?: number;
  observed_value?: number;
  webhook_status?: string;
  webhook_error?: string;
  fired_at?: string;
  acked_at?: string;
  acked_by?: string;
};

export type AutomationAction = {
  type?: string;
  webhook_url?: string;
  network?: string;
};

export type AutomationRule = {
  id?: string;
  customer_id?: string;
  campaign_id?: string;
  name?: string;
  metric?: string;
  operator?: string;
  threshold?: number;
  window_minutes?: number;
  group_by?: string;
  actions?: AutomationAction[];
  cooldown_minutes?: number;
  eval_interval_minutes?: number;
  enabled?: boolean;
  last_fired_at?: string;
  created_at?: string;
  updated_at?: string;
};

export type UpsertAutomationRuleBody = {
  customer_id: string;
  campaign_id?: string;
  name: string;
  metric: string;
  operator: string;
  threshold: number;
  window_minutes: number;
  group_by?: string;
  actions?: AutomationAction[];
  cooldown_minutes?: number;
  eval_interval_minutes?: number;
  enabled: boolean;
  preset_key?: string;
  preset_parameters?: Record<string, number>;
};

export type AutomationPreset = {
  key?: string;
  title?: string;
  description?: string;
  parameters_schema?: Array<{
    key?: string;
    type?: string;
    description?: string;
    default?: number;
  }>;
  required_license_features?: string[];
};

export type AutomationDryRunResponse = {
  would_fire?: Array<{
    rule_id?: string;
    campaign_id?: string;
    placement_id?: string;
    metric?: string;
    operator?: string;
    threshold?: number;
    observed_value?: number;
    actions?: AutomationAction[];
  }>;
};

export function buildCostSyncHistoryUrl(params: {
  customerId?: string;
  limit: number;
  offset: number;
}): string {
  const qs = new URLSearchParams({
    limit: String(params.limit),
    offset: String(params.offset),
  });
  if (params.customerId) qs.set('customer_id', params.customerId);
  return `/api/v1/cost-sync/history?${qs.toString()}`;
}

export async function fetchCostSyncNetworks(signal?: AbortSignal): Promise<CostSyncNetworkSchema[]> {
  const result = await api<CostSyncNetworkSchema[]>('/api/v1/cost-sync/networks', { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function fetchCostSyncCredentials(
  customerId: string,
  signal?: AbortSignal
): Promise<CostSyncCredential[]> {
  const qs = new URLSearchParams({ customer_id: customerId });
  const result = await api<CostSyncCredential[]>(`/api/v1/cost-sync/credentials?${qs.toString()}`, {
    signal,
  });
  return Array.isArray(result.data) ? result.data : [];
}

export async function fetchCostSyncHistory(
  params: { customerId?: string; limit: number; offset: number },
  signal?: AbortSignal
): Promise<CostSyncRun[]> {
  const result = await api<CostSyncRun[]>(buildCostSyncHistoryUrl(params), { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function upsertCostSyncCredential(
  network: string,
  body: UpsertCostSyncCredentialBody
): Promise<CostSyncCredential> {
  const result = await apiConfirmed<CostSyncCredential>(
    `/api/v1/cost-sync/credentials/${encodeURIComponent(network)}`,
    { method: 'PUT', body: JSON.stringify(body) }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('save credential failed');
  }
  return result.data ?? {};
}

export async function deleteCostSyncCredential(
  network: string,
  customerId: string
): Promise<void> {
  const qs = new URLSearchParams({ customer_id: customerId });
  const result = await apiConfirmed(
    `/api/v1/cost-sync/credentials/${encodeURIComponent(network)}?${qs.toString()}`,
    { method: 'DELETE' }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('delete credential failed');
  }
}

export async function runCostSync(body: RunCostSyncBody): Promise<void> {
  const result = await apiConfirmed('/api/v1/cost-sync/run', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('cost sync run failed');
  }
}

export async function fetchPostbackCampaignStatus(
  signal?: AbortSignal
): Promise<PostbackCampaignStatus[]> {
  const result = await api<PostbackCampaignStatus[]>('/api/v1/postbacks/campaign-status', { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function fetchPostbackDlq(signal?: AbortSignal): Promise<PostbackDlqRow[]> {
  const result = await api<PostbackDlqRow[]>('/api/v1/postbacks/dlq', { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function retryPostbackDlq(id: number): Promise<void> {
  const result = await apiConfirmed(`/api/v1/postbacks/dlq/${id}/retry`, { method: 'POST' });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('dlq retry failed');
  }
}

export async function fetchIntegrationSchemas(signal?: AbortSignal): Promise<IntegrationSchema[]> {
  const result = await api<IntegrationSchema[]>('/api/v1/integration/schemas', { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function fetchIntegrationSchema(
  id: string,
  signal?: AbortSignal
): Promise<IntegrationSchema> {
  const result = await api<IntegrationSchema>(
    `/api/v1/integration/schemas/${encodeURIComponent(id)}`,
    { signal }
  );
  return result.data ?? {};
}

export async function createIntegrationSchema(
  body: CreateIntegrationSchemaBody
): Promise<IntegrationSchema> {
  const result = await apiConfirmed<IntegrationSchema>('/api/v1/integration/schemas', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('create schema failed');
  }
  return result.data ?? {};
}

export async function applyIntegrationSchema(
  id: string,
  body: ApplyIntegrationSchemaBody
): Promise<void> {
  const result = await apiConfirmed(
    `/api/v1/integration/schemas/${encodeURIComponent(id)}/apply`,
    { method: 'POST', body: JSON.stringify(body) }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('apply schema failed');
  }
}

export async function fetchIntegrationTemplates(
  signal?: AbortSignal
): Promise<IntegrationTemplateCatalogEntry[]> {
  const result = await api<IntegrationTemplateCatalogEntry[]>('/api/v1/integration/templates', {
    signal,
  });
  return Array.isArray(result.data) ? result.data : [];
}

export async function importIntegrationTemplates(
  body: ImportIntegrationTemplatesBody
): Promise<IntegrationSchema[]> {
  const result = await apiConfirmed<IntegrationSchema[]>('/api/v1/integration/templates/import', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('import templates failed');
  }
  return Array.isArray(result.data) ? result.data : [];
}

export async function fetchSupplySellers(signal?: AbortSignal): Promise<SupplySeller[]> {
  const result = await api<SupplySeller[]>('/api/v1/supply/sellers', { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function createSupplySeller(body: SupplySellerWriteBody): Promise<SupplySeller> {
  const result = await apiConfirmed<SupplySeller>('/api/v1/supply/sellers', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('create seller failed');
  }
  return result.data ?? {};
}

export async function updateSupplySeller(
  id: number,
  body: SupplySellerWriteBody
): Promise<SupplySeller> {
  const result = await apiConfirmed<SupplySeller>(`/api/v1/supply/sellers/${id}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('update seller failed');
  }
  return result.data ?? {};
}

export async function deleteSupplySeller(id: number): Promise<void> {
  const result = await apiConfirmed(`/api/v1/supply/sellers/${id}`, { method: 'DELETE' });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('delete seller failed');
  }
}

export async function fetchSupplyAdsTxt(signal?: AbortSignal): Promise<SupplyAdsTxtEntry[]> {
  const result = await api<SupplyAdsTxtEntry[]>('/api/v1/supply/ads-txt', { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function createSupplyAdsTxt(body: SupplyAdsTxtWriteBody): Promise<SupplyAdsTxtEntry> {
  const result = await apiConfirmed<SupplyAdsTxtEntry>('/api/v1/supply/ads-txt', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('create ads.txt entry failed');
  }
  return result.data ?? {};
}

export async function updateSupplyAdsTxt(
  id: number,
  body: SupplyAdsTxtWriteBody
): Promise<SupplyAdsTxtEntry> {
  const result = await apiConfirmed<SupplyAdsTxtEntry>(`/api/v1/supply/ads-txt/${id}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('update ads.txt entry failed');
  }
  return result.data ?? {};
}

export async function deleteSupplyAdsTxt(id: number): Promise<void> {
  const result = await apiConfirmed(`/api/v1/supply/ads-txt/${id}`, { method: 'DELETE' });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('delete ads.txt entry failed');
  }
}

export async function fetchSupplyValidation(signal?: AbortSignal): Promise<SupplyValidation> {
  const result = await api<SupplyValidation>('/api/v1/supply/validation', { signal });
  return result.data ?? {};
}

export async function fetchSupplySellersPreview(signal?: AbortSignal): Promise<string> {
  const result = await api<string>('/api/v1/supply/preview/sellers.json', { signal });
  if (typeof result.data === 'string') return result.data;
  return JSON.stringify(result.data, null, 2);
}

export async function fetchSupplyAdsTxtPreview(signal?: AbortSignal): Promise<string> {
  const result = await api<string>('/api/v1/supply/preview/ads.txt', { signal });
  if (typeof result.data === 'string') return result.data;
  return String(result.data ?? '');
}

export async function fetchSupplyExportPath(signal?: AbortSignal): Promise<SupplyExportPath> {
  const result = await api<SupplyExportPath>('/api/v1/supply/export-path', { signal });
  return result.data ?? {};
}

export function buildMarginGuardPoliciesUrl(campaignId: string): string {
  const qs = new URLSearchParams({ campaign_id: campaignId });
  return `/api/v1/margin-guard/policies?${qs.toString()}`;
}

export function buildMarginGuardActivityUrl(campaignId: string): string {
  const qs = new URLSearchParams({ campaign_id: campaignId });
  return `/api/v1/margin-guard/activity?${qs.toString()}`;
}

export async function fetchMarginGuardPolicies(
  campaignId: string,
  signal?: AbortSignal
): Promise<MarginGuardPolicy[]> {
  const result = await api<MarginGuardPolicy[]>(buildMarginGuardPoliciesUrl(campaignId), { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function createMarginGuardPolicy(body: MarginGuardPolicy): Promise<MarginGuardPolicy> {
  const result = await apiConfirmed<MarginGuardPolicy>('/api/v1/margin-guard/policies', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('create policy failed');
  }
  return result.data ?? {};
}

export async function fetchMarginGuardActivity(
  campaignId: string,
  signal?: AbortSignal
): Promise<MarginGuardActivity[]> {
  const result = await api<MarginGuardActivity[]>(buildMarginGuardActivityUrl(campaignId), {
    signal,
  });
  return Array.isArray(result.data) ? result.data : [];
}

export async function removeMarginGuardOverride(body: MarginGuardOverrideBody): Promise<void> {
  const result = await apiConfirmed('/api/v1/margin-guard/overrides', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('remove override failed');
  }
}

export function buildSmartAlertRulesUrl(customerId: string): string {
  const qs = new URLSearchParams({ customer_id: customerId });
  return `/api/v1/smart-alerts/rules?${qs.toString()}`;
}

export function buildSmartAlertHistoryUrl(customerId: string, limit: number): string {
  const qs = new URLSearchParams({ customer_id: customerId, limit: String(limit) });
  return `/api/v1/smart-alerts/history?${qs.toString()}`;
}

export async function fetchSmartAlertRules(
  customerId: string,
  signal?: AbortSignal
): Promise<SmartAlertRule[]> {
  const result = await api<SmartAlertRule[]>(buildSmartAlertRulesUrl(customerId), { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function createSmartAlertRule(body: UpsertSmartAlertRuleBody): Promise<SmartAlertRule> {
  const result = await apiConfirmed<SmartAlertRule>('/api/v1/smart-alerts/rules', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('create smart alert rule failed');
  }
  return result.data ?? {};
}

export async function updateSmartAlertRule(
  id: string,
  body: UpsertSmartAlertRuleBody
): Promise<SmartAlertRule> {
  const result = await apiConfirmed<SmartAlertRule>(
    `/api/v1/smart-alerts/rules/${encodeURIComponent(id)}`,
    { method: 'PATCH', body: JSON.stringify(body) }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('update smart alert rule failed');
  }
  return result.data ?? {};
}

export async function deleteSmartAlertRule(id: string): Promise<void> {
  const result = await apiConfirmed(`/api/v1/smart-alerts/rules/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('delete smart alert rule failed');
  }
}

export async function fetchSmartAlertHistory(
  customerId: string,
  limit: number,
  signal?: AbortSignal
): Promise<SmartAlertEvent[]> {
  const result = await api<SmartAlertEvent[]>(buildSmartAlertHistoryUrl(customerId, limit), {
    signal,
  });
  return Array.isArray(result.data) ? result.data : [];
}

export async function ackSmartAlertEvent(id: string): Promise<void> {
  const result = await apiConfirmed(`/api/v1/smart-alerts/events/${encodeURIComponent(id)}/ack`, {
    method: 'POST',
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('ack event failed');
  }
}

export function buildAutomationRulesUrl(customerId: string): string {
  const qs = new URLSearchParams({ customer_id: customerId });
  return `/api/v1/automation/rules?${qs.toString()}`;
}

export async function fetchAutomationPresets(signal?: AbortSignal): Promise<AutomationPreset[]> {
  const result = await api<AutomationPreset[]>('/api/v1/automation/presets', { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function fetchAutomationRules(
  customerId: string,
  signal?: AbortSignal
): Promise<AutomationRule[]> {
  const result = await api<AutomationRule[]>(buildAutomationRulesUrl(customerId), { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function createAutomationRule(body: UpsertAutomationRuleBody): Promise<AutomationRule> {
  const result = await apiConfirmed<AutomationRule>('/api/v1/automation/rules', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('create automation rule failed');
  }
  return result.data ?? {};
}

export async function updateAutomationRule(
  id: string,
  body: UpsertAutomationRuleBody
): Promise<AutomationRule> {
  const result = await apiConfirmed<AutomationRule>(
    `/api/v1/automation/rules/${encodeURIComponent(id)}`,
    { method: 'PUT', body: JSON.stringify(body) }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('update automation rule failed');
  }
  return result.data ?? {};
}

export async function deleteAutomationRule(id: string): Promise<void> {
  const result = await apiConfirmed(`/api/v1/automation/rules/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('delete automation rule failed');
  }
}

export async function dryRunAutomationRule(id: string): Promise<AutomationDryRunResponse> {
  const result = await apiConfirmed<AutomationDryRunResponse>(
    `/api/v1/automation/rules/${encodeURIComponent(id)}/dry-run`,
    { method: 'POST' }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('dry-run failed');
  }
  return result.data ?? {};
}
