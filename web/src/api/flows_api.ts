import { apiFetch, apiJson, apiJsonArray, parseApiError } from './client.js';
import type {
  CreateFlowRequest,
  Flow,
  FlowValidateResponse,
  UpdateFlowRequest,
} from './types.js';
import type { FlowPath } from './types.js';

export async function listFlows(signal?: AbortSignal): Promise<Flow[]> {
  return apiJsonArray<Flow>('/api/v1/flows', { signal });
}

export async function getFlow(id: string, signal?: AbortSignal): Promise<Flow> {
  return apiJson<Flow>(`/api/v1/flows/${encodeURIComponent(id)}`, { signal });
}

export async function createFlow(body: CreateFlowRequest, signal?: AbortSignal): Promise<Flow> {
  return apiJson<Flow>('/api/v1/flows', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function updateFlow(
  id: string,
  body: UpdateFlowRequest,
  signal?: AbortSignal
): Promise<Flow> {
  return apiJson<Flow>(`/api/v1/flows/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(body),
    signal,
  });
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
  return apiJson<Flow>(`/api/v1/flows/${encodeURIComponent(id)}/clone`, {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function validateFlowPaths(
  paths: FlowPath[],
  signal?: AbortSignal
): Promise<FlowValidateResponse> {
  const response = await apiFetch('/api/v1/flows/validate', {
    method: 'POST',
    body: JSON.stringify({ paths }),
    signal,
  });
  const payload = (await response.json()) as FlowValidateResponse;
  if (!response.ok) {
    const message =
      payload.path_errors?.map((row) => row.message).filter(Boolean).join('; ') ||
      'Flow validation failed';
    throw new Error(message);
  }
  return payload;
}
