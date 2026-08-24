import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type LanderDTO = {
  id: string;
  name: string;
  url?: string;
  hosted_asset_id?: string | null;
  hosted_url?: string;
  created_at?: string;
};

export type OfferDTO = {
  id: string;
  name: string;
  url: string;
  created_at?: string;
};

export type FlowPathLanderRef = {
  lander_id: string;
  weight: number;
};

export type FlowPathOfferRef = {
  offer_id: string;
  weight: number;
};

export type FlowPathDTO = {
  weight: number;
  landers: FlowPathLanderRef[];
  offers: FlowPathOfferRef[];
};

export type FlowDTO = {
  id: string;
  name: string;
  paths: FlowPathDTO[] | string;
  created_at?: string;
};

export async function fetchLanders(): Promise<LanderDTO[]> {
  const res = await api<LanderDTO[]>('/api/v1/landers');
  return Array.isArray(res.data) ? res.data : [];
}

export async function createLander(name: string, url?: string): Promise<LanderDTO> {
  const res = await apiConfirmed<LanderDTO>('/api/v1/landers', {
    method: 'POST',
    body: JSON.stringify({ name, url: url ?? '' }),
  });
  return res.data;
}

/**
 * Upload a ZIP archive and publish it as the hosted lander for the given id.
 * @param landerId - Lander UUID.
 * @param zipFile - Browser File from input[type=file].
 */
export async function uploadHostedLanderZip(landerId: string, zipFile: File): Promise<LanderDTO> {
  const form = new FormData();
  form.append('zip', zipFile, zipFile.name);
  const res = await apiConfirmed<LanderDTO>(`/api/v1/landers/${landerId}/hosted-upload`, {
    method: 'POST',
    body: form,
  });
  return res.data;
}

export type HostedEditorFileDTO = {
  path: string;
  size: number;
  editable: boolean;
};

export type HostedEditorStateDTO = {
  lander_id: string;
  name: string;
  draft_version: number;
  published_version: number;
  has_unpublished_draft: boolean;
  files: HostedEditorFileDTO[];
  preview_url?: string;
};

export type HostedEditorSaveResultDTO = {
  draft_version: number;
  has_unpublished_draft: boolean;
};

/** Encode a relative lander file path for hosted-files API segments. */
export function encodeHostedEditorFilePath(relPath: string): string {
  return relPath
    .split('/')
    .filter(Boolean)
    .map((segment) => encodeURIComponent(segment))
    .join('/');
}

export async function fetchHostedEditorState(landerId: string): Promise<HostedEditorStateDTO> {
  const res = await api<HostedEditorStateDTO>(`/api/v1/landers/${landerId}/hosted-editor`);
  return res.data;
}

export async function fetchHostedEditorFile(landerId: string, relPath: string): Promise<string> {
  const encoded = encodeHostedEditorFilePath(relPath);
  const res = await api<{ content: string }>(
    `/api/v1/landers/${landerId}/hosted-files/${encoded}`
  );
  return res.data.content ?? '';
}

export async function saveHostedEditorFile(
  landerId: string,
  relPath: string,
  content: string
): Promise<HostedEditorSaveResultDTO> {
  const encoded = encodeHostedEditorFilePath(relPath);
  const res = await apiConfirmed<HostedEditorSaveResultDTO>(
    `/api/v1/landers/${landerId}/hosted-files/${encoded}`,
    {
      method: 'PUT',
      body: JSON.stringify({ content }),
    }
  );
  return res.data;
}

export async function publishHostedLanderDraft(
  landerId: string,
  version = 0
): Promise<LanderDTO> {
  const res = await apiConfirmed<LanderDTO>(`/api/v1/landers/${landerId}/hosted-publish`, {
    method: 'POST',
    body: JSON.stringify({ version }),
  });
  return res.data;
}

export async function fetchOffers(): Promise<OfferDTO[]> {
  const res = await api<OfferDTO[]>('/api/v1/offers');
  return Array.isArray(res.data) ? res.data : [];
}

export async function createOffer(name: string, url: string): Promise<OfferDTO> {
  const res = await apiConfirmed<OfferDTO>('/api/v1/offers', {
    method: 'POST',
    body: JSON.stringify({ name, url }),
  });
  return res.data;
}

export async function fetchFlows(): Promise<FlowDTO[]> {
  const res = await api<FlowDTO[]>('/api/v1/flows');
  return Array.isArray(res.data) ? res.data : [];
}

export async function createFlow(name: string, paths: FlowPathDTO[]): Promise<FlowDTO> {
  const res = await apiConfirmed<FlowDTO>('/api/v1/flows', {
    method: 'POST',
    body: JSON.stringify({ name, paths }),
  });
  return res.data;
}

export async function fetchFlow(flowId: string): Promise<FlowDTO> {
  const res = await api<FlowDTO>(`/api/v1/flows/${flowId}`);
  return res.data;
}

export async function updateFlow(
  flowId: string,
  name: string,
  paths: FlowPathDTO[]
): Promise<FlowDTO> {
  const res = await apiConfirmed<FlowDTO>(`/api/v1/flows/${flowId}`, {
    method: 'PUT',
    body: JSON.stringify({ name, paths }),
  });
  return res.data;
}

export function parseFlowPaths(raw: FlowDTO['paths']): FlowPathDTO[] {
  if (Array.isArray(raw)) return raw;
  if (typeof raw === 'string' && raw.trim()) {
    try {
      const parsed = JSON.parse(raw) as FlowPathDTO[];
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  }
  return [];
}

export function summarizeFlowPaths(paths: FlowPathDTO[]): string {
  if (!paths.length) return '-';
  const path = paths[0];
  const landerCount = path.landers?.length ?? 0;
  const offerCount = path.offers?.length ?? 0;
  const extra = paths.length > 1 ? ` (+${paths.length - 1} paths)` : '';
  return `${landerCount} lander(s), ${offerCount} offer(s)${extra}`;
}
