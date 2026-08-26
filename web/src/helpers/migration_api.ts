import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import { getOrCreate } from './idempotency.js';

export type MigrationSourceKind = 'keitaro_json' | 'binom_json' | 'native_v1';

export type MigrationSourcesResponse = {
  sources: Array<{ kind: MigrationSourceKind; label: string }>;
  max_payload_bytes: number;
};

export type MigrationWarning = {
  slug: string;
  message: string;
  campaign_ref?: string;
};

export type MappedMigrationCampaign = {
  ref: string;
  name: string;
  traffic_source_name?: string;
  bundled_slug?: string;
  ui_template_id?: string;
  integration_schema_name?: string;
  click_query_params?: Record<string, string>;
  target_url?: string;
  budget_limit_micro?: number;
  ingress_cost_param?: string;
  postback_url_template?: string;
};

export type MigrationPreviewResult = {
  source_kind: MigrationSourceKind;
  mapped_campaigns: MappedMigrationCampaign[];
  warnings?: MigrationWarning[];
  secrets_stripped: number;
};

export type MigrationImportFailure = {
  ref: string;
  name?: string;
  message: string;
};

export type MigrationImportResult = {
  import_batch_id: string;
  imported: Array<{ id: string; name: string }>;
  warnings?: MigrationWarning[];
  failed?: MigrationImportFailure[];
};

/**
 * Lists supported external migration source kinds.
 */
export async function fetchMigrationSources(): Promise<MigrationSourcesResponse> {
  const res = await api<MigrationSourcesResponse>('/api/v1/campaigns/migrate/sources');
  return res.data;
}

/**
 * Parses a foreign export payload without creating campaigns.
 */
export async function previewMigration(
  sourceKind: MigrationSourceKind,
  payload: unknown
): Promise<MigrationPreviewResult> {
  const res = await api<MigrationPreviewResult>('/api/v1/campaigns/migrate/preview', {
    method: 'POST',
    body: JSON.stringify({ source_kind: sourceKind, payload }),
  });
  return res.data;
}

/**
 * Imports mapped campaigns from a foreign tracker export.
 */
export async function importMigration(
  customerId: string,
  sourceKind: MigrationSourceKind,
  payload: unknown,
  opts?: { namePrefix?: string; budgetLimitMicro?: number }
): Promise<MigrationImportResult> {
  const key = getOrCreate(`migrate:${customerId}:${sourceKind}`);
  const res = await apiConfirmed<MigrationImportResult>('/api/v1/campaigns/migrate/import', {
    method: 'POST',
    headers: { 'Idempotency-Key': key },
    body: JSON.stringify({
      customer_id: customerId,
      source_kind: sourceKind,
      payload,
      name_prefix: opts?.namePrefix ?? '',
      budget_limit_micro: opts?.budgetLimitMicro,
    }),
  });
  return res.data;
}
