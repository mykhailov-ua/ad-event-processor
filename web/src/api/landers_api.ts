import { apiFetch, apiJson } from './client.js';
import type { CreateLanderRequest, HostedEditorState, Lander } from './types.js';

export async function listLanders(signal?: AbortSignal): Promise<Lander[]> {
  return apiJson<Lander[]>('/api/v1/landers', { signal });
}

export async function createLander(body: CreateLanderRequest, signal?: AbortSignal): Promise<Lander> {
  return apiJson<Lander>('/api/v1/landers', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function getHostedEditorState(
  landerId: string,
  signal?: AbortSignal,
): Promise<HostedEditorState> {
  return apiJson<HostedEditorState>(
    `/api/v1/landers/${encodeURIComponent(landerId)}/hosted-editor`,
    { signal },
  );
}

export async function uploadHostedLanderFiles(
  landerId: string,
  zipFile: File,
  signal?: AbortSignal,
): Promise<Lander> {
  const form = new FormData();
  form.append('zip', zipFile);
  const response = await apiFetch(
    `/api/v1/landers/${encodeURIComponent(landerId)}/hosted-upload`,
    {
      method: 'POST',
      body: form,
      signal,
    },
  );
  if (!response.ok) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
  return (await response.json()) as Lander;
}

export async function getHostedLanderFile(
  landerId: string,
  filePath: string,
  signal?: AbortSignal,
): Promise<string> {
  const search = new URLSearchParams({ path: filePath });
  const response = await apiFetch(
    `/api/v1/landers/${encodeURIComponent(landerId)}/hosted-files?${search.toString()}`,
    { signal },
  );
  if (!response.ok) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
  return response.text();
}

export async function putHostedLanderFile(
  landerId: string,
  filePath: string,
  content: string,
  signal?: AbortSignal,
): Promise<void> {
  const response = await apiFetch(
    `/api/v1/landers/${encodeURIComponent(landerId)}/hosted-files`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: filePath, content }),
      signal,
    },
  );
  if (!response.ok) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}

export async function publishHostedLander(landerId: string, signal?: AbortSignal): Promise<Lander> {
  return apiJson<Lander>(
    `/api/v1/landers/${encodeURIComponent(landerId)}/hosted-publish`,
    { method: 'POST', signal },
  );
}
