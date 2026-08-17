import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import type {
  ApplyCampaignTemplatesResult,
  CreateIntegrationSchemaBody,
  IntegrationSchemaDTO,
  IntegrationTemplateCatalogEntry,
} from '../types/api/integration.js';

/** Bundled traffic-source template names (GM-M4 catalog). */
export const BUNDLED_TRAFFIC_TEMPLATES = [
  { value: 'traffic_propellerads', label: 'PropellerAds' },
  { value: 'traffic_exoclick', label: 'ExoClick' },
  { value: 'traffic_facebook', label: 'Facebook / Meta' },
] as const;

/** Bundled affiliate network template names (GM-M4 catalog). */
export const BUNDLED_AFFILIATE_TEMPLATES = [
  { value: 'affiliate_everad', label: 'Everad' },
  { value: 'affiliate_leadbit', label: 'Leadbit' },
] as const;

export type IntegrationSchemaKind = 'inbound_tokens' | 'outbound_postback' | 'status_mapping';

/** Starter documents for POST /integration/schemas (inner schema body). */
export const INTEGRATION_SCHEMA_STARTERS: Record<IntegrationSchemaKind, Record<string, unknown>> = {
  inbound_tokens: {
    version: 1,
    tokens: [
      { name: 'gclid', query_key: 'gclid', max_len: 256 },
      { name: 'sub1', query_key: 'sub1', max_len: 128 },
    ],
    macros: [
      { name: 'campaign_id', source: 'query', key: 'campaign_id', required: true },
    ],
  },
  outbound_postback: {
    version: 1,
    url_template: 'https://aff.example.com/postback?click_id={click_id}&payout={payout}&status={status}&sub1={sub1}',
    placeholders: ['click_id', 'payout', 'currency', 'status', 'sub1'],
  },
  status_mapping: {
    version: 1,
    status_map: {
      approved: 'conversion',
      rejected: 'rejected',
      pending: 'pending',
    },
  },
};

/**
 * List integration schemas stored in Postgres.
 */
export async function fetchIntegrationSchemas(): Promise<IntegrationSchemaDTO[]> {
  const { data } = await api<IntegrationSchemaDTO[]>('/api/v1/integration/schemas');
  return data ?? [];
}

/**
 * Fetch a single integration schema by id.
 */
export async function fetchIntegrationSchema(schemaId: string): Promise<IntegrationSchemaDTO> {
  const { data } = await api<IntegrationSchemaDTO>(
    `/api/v1/integration/schemas/${encodeURIComponent(schemaId)}`,
  );
  return data ?? {
    id: schemaId,
    name: '',
    version: 0,
    kind: '',
    schema: null,
    created_at: '',
    updated_at: '',
  };
}

/**
 * Create a custom integration schema (JSON/YAML body validated server-side).
 */
export async function createIntegrationSchema(
  body: CreateIntegrationSchemaBody,
): Promise<IntegrationSchemaDTO> {
  const res = await apiConfirmed<IntegrationSchemaDTO>('/api/v1/integration/schemas', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  return res.data ?? {
    id: '',
    name: body.name,
    version: body.version,
    kind: '',
    schema: body.schema,
    created_at: '',
    updated_at: '',
  };
}

/**
 * List bundled YAML templates available for import.
 */
export async function fetchBundledTemplates(): Promise<IntegrationTemplateCatalogEntry[]> {
  const { data } = await api<IntegrationTemplateCatalogEntry[]>('/api/v1/integration/templates');
  return data ?? [];
}

/**
 * Import selected bundled templates into integration_schemas.
 */
export async function importBundledTemplates(names: string[]): Promise<IntegrationSchemaDTO[]> {
  const res = await apiConfirmed<IntegrationSchemaDTO[]>('/api/v1/integration/templates/import', {
    method: 'POST',
    body: JSON.stringify({ names }),
  });
  return res.data ?? [];
}

/**
 * Apply an integration schema to a campaign (postback, inbound tokens, or status mapping).
 */
export async function applyIntegrationSchema(
  schemaId: string,
  campaignId: string,
): Promise<Record<string, string>> {
  const res = await apiConfirmed<Record<string, string>>(
    `/api/v1/integration/schemas/${encodeURIComponent(schemaId)}/apply`,
    {
      method: 'POST',
      body: JSON.stringify({ campaign_id: campaignId }),
    },
  );
  return res.data ?? {};
}

/**
 * Apply GM-M4 bundled traffic / affiliate templates to a campaign.
 */
export async function applyCampaignTemplates(
  campaignId: string,
  body: {
    traffic_source?: string;
    affiliate_network?: string;
    tracking_domain?: string;
  },
): Promise<ApplyCampaignTemplatesResult> {
  const res = await apiConfirmed<ApplyCampaignTemplatesResult>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/apply-templates`,
    {
      method: 'POST',
      body: JSON.stringify(body),
    },
  );
  return res.data ?? { campaign_id: campaignId };
}
