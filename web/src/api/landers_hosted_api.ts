import { apiFetch, apiJsonValidated, parseApiError } from './client.js';
import {
  parseHostedEditorFileBody,
  parseHostedEditorSaveResult,
  parseHostedEditorState,
  parseLander,
} from './validate.js';
import type {
  HostedEditorFileBody,
  HostedEditorSaveResult,
  HostedEditorState,
  Lander,
} from './types.js';

export async function getHostedEditorState(
  landerId: string,
  signal?: AbortSignal
): Promise<HostedEditorState> {
  return apiJsonValidated(
    `/api/v1/landers/${encodeURIComponent(landerId)}/hosted-editor`,
    { signal },
    parseHostedEditorState
  );
}

export async function readHostedEditorFile(
  landerId: string,
  filePath: string,
  signal?: AbortSignal
): Promise<HostedEditorFileBody> {
  const encoded = filePath
    .split('/')
    .map((part) => encodeURIComponent(part))
    .join('/');
  return apiJsonValidated(
    `/api/v1/landers/${encodeURIComponent(landerId)}/hosted-files/${encoded}`,
    { signal },
    parseHostedEditorFileBody
  );
}

export async function saveHostedEditorFile(
  landerId: string,
  filePath: string,
  content: string,
  signal?: AbortSignal
): Promise<HostedEditorSaveResult> {
  const encoded = filePath
    .split('/')
    .map((part) => encodeURIComponent(part))
    .join('/');
  return apiJsonValidated(
    `/api/v1/landers/${encodeURIComponent(landerId)}/hosted-files/${encoded}`,
    {
      method: 'PUT',
      body: JSON.stringify({ content }),
      signal,
    },
    parseHostedEditorSaveResult
  );
}

export async function publishHostedLander(
  landerId: string,
  version: number,
  signal?: AbortSignal
): Promise<Lander> {
  return apiJsonValidated(
    `/api/v1/landers/${encodeURIComponent(landerId)}/hosted-publish`,
    {
      method: 'POST',
      body: JSON.stringify({ version }),
      signal,
    },
    parseLander
  );
}

export async function uploadHostedLanderZip(
  landerId: string,
  file: File,
  signal?: AbortSignal
): Promise<Lander> {
  const form = new FormData();
  form.append('zip', file);
  const response = await apiFetch(`/api/v1/landers/${encodeURIComponent(landerId)}/hosted-upload`, {
    method: 'POST',
    body: form,
    signal,
  });
  if (!response.ok) {
    throw await parseApiError(response);
  }
  return parseLander(await response.json());
}
