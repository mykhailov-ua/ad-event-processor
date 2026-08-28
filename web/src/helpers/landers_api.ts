import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type HostedEditorFile = {
  path?: string;
  size?: number;
  editable?: boolean;
};

export type HostedEditorState = {
  lander_id?: string;
  name?: string;
  draft_version?: number;
  published_version?: number;
  has_unpublished_draft?: boolean;
  preview_url?: string;
  files?: HostedEditorFile[];
};

export type LanderEditorTab = 'editor' | 'files' | 'publish';

export const LANDER_EDITOR_TABS: Array<{ id: LanderEditorTab; label: string }> = [
  { id: 'editor', label: 'Editor' },
  { id: 'files', label: 'Files' },
  { id: 'publish', label: 'Publish' },
];

export function parseLanderEditorTab(raw: string | null): LanderEditorTab {
  const allowed: LanderEditorTab[] = ['editor', 'files', 'publish'];
  return allowed.includes(raw as LanderEditorTab) ? (raw as LanderEditorTab) : 'editor';
}

export async function fetchHostedEditorState(
  landerId: string,
  signal?: AbortSignal
): Promise<HostedEditorState> {
  const result = await api<HostedEditorState>(
    `/api/v1/landers/${encodeURIComponent(landerId)}/hosted-editor`,
    { signal }
  );
  return result.data ?? {};
}

export async function fetchHostedFile(
  landerId: string,
  path: string,
  signal?: AbortSignal
): Promise<string> {
  const segments = path.split('/').map((seg) => encodeURIComponent(seg)).join('/');
  const result = await api<{ content?: string }>(
    `/api/v1/landers/${encodeURIComponent(landerId)}/hosted-files/${segments}`,
    { signal }
  );
  return result.data?.content ?? '';
}

export async function saveHostedFile(
  landerId: string,
  path: string,
  content: string
): Promise<void> {
  const segments = path.split('/').map((seg) => encodeURIComponent(seg)).join('/');
  const result = await apiConfirmed(
    `/api/v1/landers/${encodeURIComponent(landerId)}/hosted-files/${segments}`,
    { method: 'PUT', body: JSON.stringify({ content }) }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('save file failed');
  }
}

export async function publishHostedLander(
  landerId: string,
  version?: number
): Promise<void> {
  const body = version != null ? { version } : {};
  const result = await apiConfirmed(
    `/api/v1/landers/${encodeURIComponent(landerId)}/hosted-publish`,
    { method: 'POST', body: JSON.stringify(body) }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('publish failed');
  }
}
