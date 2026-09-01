import { apiFetch, apiJson } from './client.js';
import type {
  AffiliateStatusPreset,
  ApplyIntegrationSchemaRequest,
  ApplyIntegrationSchemaResponse,
  CostSyncCredential,
  CostSyncCredentialsQuery,
  CostSyncHistoryQuery,
  CostSyncNetworkSchema,
  CostSyncRun,
  CostSyncSnapshot,
  CreateIntegrationSchemaRequest,
  ImportIntegrationTemplatesRequest,
  IntegrationSchema,
  IntegrationSnapshot,
  IntegrationTemplateCatalogEntry,
  PlatformCampaignLink,
  PlatformCampaignLinksQuery,
  PlatformCampaignMutation,
  PlatformCampaignMutationRequest,
  PlatformCampaignSyncRunRequest,
  PostbackCampaignStatus,
  PostbackConfig,
  PostbackDlqEntry,
  PostbackDryRunResult,
  PostbacksSnapshot,
  RunCostSyncAcceptedResponse,
  RunCostSyncRequest,
  StatusOKResponse,
  UpdatePostbackConfigRequest,
  UpsertCostSyncCredentialRequest,
  UpsertPlatformCampaignLinkRequest,
} from './types.js';

export async function listCostSyncNetworks(
  signal?: AbortSignal,
): Promise<CostSyncNetworkSchema[]> {
  return apiJson<CostSyncNetworkSchema[]>('/api/v1/cost-sync/networks', { signal });
}

export async function listCostSyncCredentials(
  params: CostSyncCredentialsQuery = {},
  signal?: AbortSignal,
): Promise<CostSyncCredential[]> {
  const search = new URLSearchParams();
  if (params.customer_id) {
    search.set('customer_id', params.customer_id);
  }
  const query = search.toString();
  const path = query
    ? `/api/v1/cost-sync/credentials?${query}`
    : '/api/v1/cost-sync/credentials';
  return apiJson<CostSyncCredential[]>(path, { signal });
}

export async function listCostSyncHistory(
  params: CostSyncHistoryQuery = {},
  signal?: AbortSignal,
): Promise<CostSyncRun[]> {
  const search = new URLSearchParams();
  if (params.customer_id) {
    search.set('customer_id', params.customer_id);
  }
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }
  const query = search.toString();
  const path = query ? `/api/v1/cost-sync/history?${query}` : '/api/v1/cost-sync/history';
  return apiJson<CostSyncRun[]>(path, { signal });
}

export async function runCostSync(
  body: RunCostSyncRequest,
  signal?: AbortSignal,
): Promise<RunCostSyncAcceptedResponse> {
  return apiJson<RunCostSyncAcceptedResponse>('/api/v1/cost-sync/run', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function upsertCostSyncCredential(
  network: string,
  body: UpsertCostSyncCredentialRequest,
  signal?: AbortSignal,
): Promise<CostSyncCredential> {
  return apiJson<CostSyncCredential>(
    `/api/v1/cost-sync/credentials/${encodeURIComponent(network)}`,
    {
      method: 'PUT',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export async function deleteCostSyncCredential(
  network: string,
  customerId: string,
  signal?: AbortSignal,
): Promise<void> {
  const search = new URLSearchParams({ customer_id: customerId });
  const response = await apiFetch(
    `/api/v1/cost-sync/credentials/${encodeURIComponent(network)}?${search.toString()}`,
    { method: 'DELETE', signal },
  );
  if (!response.ok && response.status !== 204) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}

export async function fetchPostbacksSnapshot(signal?: AbortSignal): Promise<PostbacksSnapshot> {
  return apiJson<PostbacksSnapshot>('/api/v1/postbacks/snapshot', { signal });
}

export async function fetchCostSyncSnapshot(
  params: { customer_id?: string; limit?: number } = {},
  signal?: AbortSignal,
): Promise<CostSyncSnapshot> {
  const search = new URLSearchParams();
  if (params.customer_id) {
    search.set('customer_id', params.customer_id);
  }
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  const query = search.toString();
  const path = query ? `/api/v1/cost-sync/snapshot?${query}` : '/api/v1/cost-sync/snapshot';
  return apiJson<CostSyncSnapshot>(path, { signal });
}

export async function fetchIntegrationSnapshot(signal?: AbortSignal): Promise<IntegrationSnapshot> {
  return apiJson<IntegrationSnapshot>('/api/v1/integration/snapshot', { signal });
}

export async function listPostbackConfigs(signal?: AbortSignal): Promise<PostbackConfig[]> {
  return apiJson<PostbackConfig[]>('/api/v1/postbacks/config', { signal });
}

export async function listPostbackDlq(signal?: AbortSignal): Promise<PostbackDlqEntry[]> {
  return apiJson<PostbackDlqEntry[]>('/api/v1/postbacks/dlq', { signal });
}

export async function listPostbackCampaignStatus(
  signal?: AbortSignal,
): Promise<PostbackCampaignStatus[]> {
  return apiJson<PostbackCampaignStatus[]>('/api/v1/postbacks/campaign-status', { signal });
}

export async function retryPostbackDlq(id: string, signal?: AbortSignal): Promise<StatusOKResponse> {
  return apiJson<StatusOKResponse>(`/api/v1/postbacks/dlq/${encodeURIComponent(id)}/retry`, {
    method: 'POST',
    signal,
  });
}

export async function updatePostbackConfig(
  campaignId: string,
  body: UpdatePostbackConfigRequest,
  signal?: AbortSignal,
): Promise<StatusOKResponse> {
  return apiJson<StatusOKResponse>(
    `/api/v1/postbacks/config/${encodeURIComponent(campaignId)}`,
    {
      method: 'PUT',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export async function testPostbackConfig(
  campaignId: string,
  signal?: AbortSignal,
): Promise<PostbackDryRunResult> {
  const response = await apiFetch(
    `/api/v1/postbacks/config/${encodeURIComponent(campaignId)}/test`,
    { method: 'POST', signal },
  );
  const body = (await response.json()) as PostbackDryRunResult;
  if (response.ok || response.status === 422) {
    return body;
  }
  throw new Error(response.statusText || `HTTP ${response.status}`);
}

export async function listIntegrationSchemas(
  signal?: AbortSignal,
): Promise<IntegrationSchema[]> {
  return apiJson<IntegrationSchema[]>('/api/v1/integration/schemas', { signal });
}

export async function getIntegrationSchema(
  id: string,
  signal?: AbortSignal,
): Promise<IntegrationSchema> {
  return apiJson<IntegrationSchema>(
    `/api/v1/integration/schemas/${encodeURIComponent(id)}`,
    { signal },
  );
}

export async function createIntegrationSchema(
  body: CreateIntegrationSchemaRequest,
  signal?: AbortSignal,
): Promise<IntegrationSchema> {
  return apiJson<IntegrationSchema>('/api/v1/integration/schemas', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function applyIntegrationSchema(
  id: string,
  body: ApplyIntegrationSchemaRequest,
  signal?: AbortSignal,
): Promise<ApplyIntegrationSchemaResponse> {
  return apiJson<ApplyIntegrationSchemaResponse>(
    `/api/v1/integration/schemas/${encodeURIComponent(id)}/apply`,
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export async function importIntegrationTemplates(
  body: ImportIntegrationTemplatesRequest = {},
  signal?: AbortSignal,
): Promise<IntegrationSchema[]> {
  return apiJson<IntegrationSchema[]>('/api/v1/integration/templates/import', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function listIntegrationTemplates(
  signal?: AbortSignal,
): Promise<IntegrationTemplateCatalogEntry[]> {
  return apiJson<IntegrationTemplateCatalogEntry[]>('/api/v1/integration/templates', { signal });
}

export async function listAffiliateStatusPresets(
  signal?: AbortSignal,
): Promise<AffiliateStatusPreset[]> {
  return apiJson<AffiliateStatusPreset[]>('/api/v1/integration/affiliate-status-presets', {
    signal,
  });
}

export async function listPlatformCampaignLinks(
  params: PlatformCampaignLinksQuery = {},
  signal?: AbortSignal,
): Promise<PlatformCampaignLink[]> {
  const search = new URLSearchParams();
  if (params.customer_id) {
    search.set('customer_id', params.customer_id);
  }
  if (params.campaign_id) {
    search.set('campaign_id', params.campaign_id);
  }
  const query = search.toString();
  const path = query
    ? `/api/v1/platform-campaigns/links?${query}`
    : '/api/v1/platform-campaigns/links';
  return apiJson<PlatformCampaignLink[]>(path, { signal });
}

export async function runPlatformCampaignSync(
  body: PlatformCampaignSyncRunRequest,
  signal?: AbortSignal,
): Promise<void> {
  const response = await apiFetch('/api/v1/platform-campaigns/sync-run', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
  if (!response.ok && response.status !== 204) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}

function platformCampaignLinkPath(campaignId: string, network: string): string {
  return `/api/v1/platform-campaigns/links/${encodeURIComponent(campaignId)}/${encodeURIComponent(network)}`;
}

export async function upsertPlatformCampaignLink(
  campaignId: string,
  network: string,
  body: UpsertPlatformCampaignLinkRequest,
  signal?: AbortSignal,
): Promise<PlatformCampaignLink> {
  return apiJson<PlatformCampaignLink>(platformCampaignLinkPath(campaignId, network), {
    method: 'PUT',
    body: JSON.stringify(body),
    signal,
  });
}

export async function deletePlatformCampaignLink(
  campaignId: string,
  network: string,
  signal?: AbortSignal,
): Promise<void> {
  const response = await apiFetch(platformCampaignLinkPath(campaignId, network), {
    method: 'DELETE',
    signal,
  });
  if (!response.ok && response.status !== 204) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}

export async function refreshPlatformCampaignLink(
  campaignId: string,
  network: string,
  signal?: AbortSignal,
): Promise<PlatformCampaignLink> {
  return apiJson<PlatformCampaignLink>(
    `${platformCampaignLinkPath(campaignId, network)}/refresh`,
    { method: 'POST', signal },
  );
}

export async function pausePlatformCampaign(
  campaignId: string,
  body: PlatformCampaignMutationRequest,
  signal?: AbortSignal,
): Promise<PlatformCampaignMutation> {
  return apiJson<PlatformCampaignMutation>(
    `/api/v1/platform-campaigns/${encodeURIComponent(campaignId)}/pause`,
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export async function resumePlatformCampaign(
  campaignId: string,
  body: PlatformCampaignMutationRequest,
  signal?: AbortSignal,
): Promise<PlatformCampaignMutation> {
  return apiJson<PlatformCampaignMutation>(
    `/api/v1/platform-campaigns/${encodeURIComponent(campaignId)}/resume`,
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
  );
}

export async function setPlatformCampaignBudget(
  campaignId: string,
  body: PlatformCampaignMutationRequest,
  signal?: AbortSignal,
): Promise<PlatformCampaignMutation> {
  return apiJson<PlatformCampaignMutation>(
    `/api/v1/platform-campaigns/${encodeURIComponent(campaignId)}/budget`,
    {
      method: 'PUT',
      body: JSON.stringify(body),
      signal,
    },
  );
}
