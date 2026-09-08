import { apiFetch, apiJson, apiJsonArray, parseApiError } from './client.js';
import type {
  CreateLanderRequest,
  HostedEditorFileBody,
  HostedEditorState,
  Lander,
  UpdateLanderRequest,
} from './types.js';

function encodeHostedFilePath(filePath: string): string {
  return filePath
    .split('/')
    .filter((segment) => segment.length > 0)
    .map(encodeURIComponent)
    .join('/');
}

export async function listLanders(signal?: AbortSignal): Promise<Lander[]> {
  return apiJsonArray<Lander>('/api/v1/landers', { signal });
}

export async function getLander(landerId: string, signal?: AbortSignal): Promise<Lander> {
  return apiJson<Lander>(`/api/v1/landers/${encodeURIComponent(landerId)}`, { signal });
}

export async function createLander(
  body: CreateLanderRequest,
  signal?: AbortSignal
): Promise<Lander> {
  return apiJson<Lander>('/api/v1/landers', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function updateLander(
  landerId: string,
  body: UpdateLanderRequest,
  signal?: AbortSignal
): Promise<Lander> {
  return apiJson<Lander>(`/api/v1/landers/${encodeURIComponent(landerId)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
    signal,
  });
}

export async function deleteLander(landerId: string, signal?: AbortSignal): Promise<void> {
  const response = await apiFetch(`/api/v1/landers/${encodeURIComponent(landerId)}`, {
    method: 'DELETE',
    signal,
  });
  if (!response.ok) {
    throw await parseApiError(response);
  }
}

export async function getHostedEditorState(
  landerId: string,
  signal?: AbortSignal
): Promise<HostedEditorState> {
  return apiJson<HostedEditorState>(
    `/api/v1/landers/${encodeURIComponent(landerId)}/hosted-editor`,
    { signal }
  );
}

export async function uploadHostedLanderFiles(
  landerId: string,
  zipFile: File,
  signal?: AbortSignal
): Promise<Lander> {
  const form = new FormData();
  form.append('zip', zipFile);
  const response = await apiFetch(`/api/v1/landers/${encodeURIComponent(landerId)}/hosted-upload`, {
    method: 'POST',
    body: form,
    signal,
  });
  if (!response.ok) {
    throw await parseApiError(response);
  }
  return (await response.json()) as Lander;
}

export async function getHostedLanderFile(
  landerId: string,
  filePath: string,
  signal?: AbortSignal
): Promise<string> {
  const encodedPath = encodeHostedFilePath(filePath);
  const body = await apiJson<HostedEditorFileBody>(
    `/api/v1/landers/${encodeURIComponent(landerId)}/hosted-files/${encodedPath}`,
    { signal }
  );
  return body.content;
}

export async function putHostedLanderFile(
  landerId: string,
  filePath: string,
  content: string,
  signal?: AbortSignal
): Promise<void> {
  const encodedPath = encodeHostedFilePath(filePath);
  const response = await apiFetch(
    `/api/v1/landers/${encodeURIComponent(landerId)}/hosted-files/${encodedPath}`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content }),
      signal,
    }
  );
  if (!response.ok) {
    throw await parseApiError(response);
  }
}

export async function publishHostedLander(landerId: string, signal?: AbortSignal): Promise<Lander> {
  return apiJson<Lander>(`/api/v1/landers/${encodeURIComponent(landerId)}/hosted-publish`, {
    method: 'POST',
    body: JSON.stringify({}),
    signal,
  });
}
