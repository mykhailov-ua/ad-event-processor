import { apiFetch, apiJson } from './client.js';
import type {
  CreateSavedViewRequest,
  SavedView,
  SavedViewsListQuery,
  UpdateSavedViewRequest,
} from './types.js';

export async function listSavedViews(
  params: SavedViewsListQuery,
  signal?: AbortSignal,
): Promise<SavedView[]> {
  const search = new URLSearchParams();
  search.set('customer_id', params.customer_id);
  return apiJson<SavedView[]>(`/api/v1/views?${search.toString()}`, { signal });
}

export async function getSavedView(id: string, signal?: AbortSignal): Promise<SavedView> {
  return apiJson<SavedView>(`/api/v1/views/${encodeURIComponent(id)}`, { signal });
}

export async function createSavedView(
  body: CreateSavedViewRequest,
  signal?: AbortSignal,
): Promise<SavedView> {
  return apiJson<SavedView>('/api/v1/views', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function updateSavedView(
  id: string,
  body: UpdateSavedViewRequest,
  signal?: AbortSignal,
): Promise<SavedView> {
  return apiJson<SavedView>(`/api/v1/views/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(body),
    signal,
  });
}

export async function deleteSavedView(id: string, signal?: AbortSignal): Promise<void> {
  const response = await apiFetch(`/api/v1/views/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    signal,
  });
  if (!response.ok && response.status !== 204) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}
