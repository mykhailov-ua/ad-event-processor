import { apiJson } from './client.js';
import type { CreateFlowRequest, Flow, UpdateFlowRequest } from './types.js';

export async function listFlows(signal?: AbortSignal): Promise<Flow[]> {
  return apiJson<Flow[]>('/api/v1/flows', { signal });
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
  signal?: AbortSignal,
): Promise<Flow> {
  return apiJson<Flow>(`/api/v1/flows/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(body),
    signal,
  });
}
