import { apiFetch, apiJsonValidated, parseApiError } from './client.js';
import { parseCampaignFlowValidateResponse, parseFlow, parseFlowList } from './validate.js';
import type {
  CampaignFlowValidateResponse,
  CreateFlowRequest,
  Flow,
  UpdateFlowRequest,
} from './types.js';
import type { FlowPath } from './types.js';

export async function listFlows(signal?: AbortSignal): Promise<Flow[]> {
  return apiJsonValidated('/api/v1/flows', { signal }, parseFlowList);
}

export async function getFlow(id: string, signal?: AbortSignal): Promise<Flow> {
  return apiJsonValidated(`/api/v1/flows/${encodeURIComponent(id)}`, { signal }, parseFlow);
}

export async function createFlow(body: CreateFlowRequest, signal?: AbortSignal): Promise<Flow> {
  return apiJsonValidated(
    '/api/v1/flows',
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
    parseFlow
  );
}

export async function updateFlow(
  id: string,
  body: UpdateFlowRequest,
  signal?: AbortSignal
): Promise<Flow> {
  return apiJsonValidated(
    `/api/v1/flows/${encodeURIComponent(id)}`,
    {
      method: 'PUT',
      body: JSON.stringify(body),
      signal,
    },
    parseFlow
  );
}

export async function deleteFlow(id: string, signal?: AbortSignal): Promise<void> {
  const response = await apiFetch(`/api/v1/flows/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    signal,
  });
  if (!response.ok) {
    throw await parseApiError(response);
  }
}

export async function cloneFlow(
  id: string,
  body: { name?: string },
  signal?: AbortSignal
): Promise<Flow> {
  return apiJsonValidated(
    `/api/v1/flows/${encodeURIComponent(id)}/clone`,
    {
      method: 'POST',
      body: JSON.stringify(body),
      signal,
    },
    parseFlow
  );
}

export async function validateFlowPaths(
  paths: FlowPath[],
  signal?: AbortSignal
): Promise<CampaignFlowValidateResponse> {
  const response = await apiFetch('/api/v1/flows/validate', {
    method: 'POST',
    body: JSON.stringify({ paths }),
    signal,
  });

  if (response.ok) {
    return parseCampaignFlowValidateResponse(await response.json());
  }

  if (response.status === 400) {
    const parsed = parseCampaignFlowValidateResponse(await response.json());
    const message =
      parsed.path_errors
        ?.map((row) => row.message)
        .filter(Boolean)
        .join('; ') || 'Flow validation failed';
    throw new Error(message);
  }

  throw await parseApiError(response);
}
