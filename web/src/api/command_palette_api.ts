import { apiFetch, apiJson } from './client.js';
import type {
  CommandPaletteItem,
  CommandPaletteOpenRequest,
  CommandPaletteRecentsResponse,
  CommandPaletteRoutesResponse,
  CommandPaletteSearchQuery,
  CommandPaletteSearchResponse,
  CommandPaletteRecordRecentRequest,
} from './types.js';

export async function searchCommandPalette(
  params: CommandPaletteSearchQuery,
  signal?: AbortSignal,
): Promise<CommandPaletteSearchResponse> {
  const search = new URLSearchParams();
  search.set('customer_id', params.customer_id);
  if (params.q) {
    search.set('q', params.q);
  }
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.kinds?.length) {
    for (const kind of params.kinds) {
      search.append('kinds', kind);
    }
  }
  return apiJson<CommandPaletteSearchResponse>(
    `/api/v1/command-palette/search?${search.toString()}`,
    { signal },
  );
}

export async function listCommandPaletteRoutes(
  signal?: AbortSignal,
): Promise<CommandPaletteRoutesResponse> {
  return apiJson<CommandPaletteRoutesResponse>('/api/v1/command-palette/routes', { signal });
}

export async function recordCommandPaletteOpen(
  body: CommandPaletteOpenRequest = {},
  signal?: AbortSignal,
): Promise<{ status?: string }> {
  return apiJson<{ status?: string }>('/api/v1/command-palette/open', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function listCommandPaletteRecents(
  customerId: string,
  signal?: AbortSignal,
): Promise<CommandPaletteRecentsResponse> {
  const search = new URLSearchParams({ customer_id: customerId });
  return apiJson<CommandPaletteRecentsResponse>(
    `/api/v1/command-palette/recents?${search.toString()}`,
    { signal },
  );
}

export async function recordCommandPaletteRecent(
  body: CommandPaletteRecordRecentRequest,
  signal?: AbortSignal,
): Promise<void> {
  const response = await apiFetch('/api/v1/command-palette/recents', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
  if (!response.ok && response.status !== 204) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}

export type { CommandPaletteItem };
