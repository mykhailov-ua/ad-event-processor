import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

/**
 * List sellers from the supply API.
 */
export async function fetchSellers(): Promise<unknown[]> {
  const { data } = await api('/api/v1/supply/sellers');
  return (data as unknown[] | null | undefined) ?? [];
}

/**
 * Create a seller.
 */
export async function createSeller(body: Record<string, unknown>): Promise<unknown> {
  const res = await apiConfirmed('/api/v1/supply/sellers', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  return res.data;
}

/**
 * Update a seller by id.
 */
export async function updateSeller(id: number, body: Record<string, unknown>): Promise<unknown> {
  const res = await apiConfirmed(`/api/v1/supply/sellers/${id}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
  return res.data;
}

/**
 * Delete a seller by id.
 */
export async function deleteSeller(id: number): Promise<void> {
  await apiConfirmed(`/api/v1/supply/sellers/${id}`, { method: 'DELETE' });
}

/**
 * List ads.txt entries.
 */
export async function fetchAdsTxtEntries(): Promise<unknown[]> {
  const { data } = await api('/api/v1/supply/ads-txt');
  return (data as unknown[] | null | undefined) ?? [];
}

/**
 * Create an ads.txt entry.
 */
export async function createAdsTxtEntry(body: Record<string, unknown>): Promise<unknown> {
  const res = await apiConfirmed('/api/v1/supply/ads-txt', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  return res.data;
}

/**
 * Update an ads.txt entry by id.
 */
export async function updateAdsTxtEntry(id: number, body: Record<string, unknown>): Promise<unknown> {
  const res = await apiConfirmed(`/api/v1/supply/ads-txt/${id}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
  return res.data;
}

/**
 * Delete an ads.txt entry by id.
 */
export async function deleteAdsTxtEntry(id: number): Promise<void> {
  await apiConfirmed(`/api/v1/supply/ads-txt/${id}`, { method: 'DELETE' });
}

/**
 * Fetch sellers.json preview text (pretty-printed when valid JSON).
 */
export async function fetchSellersJSONPreview(): Promise<string> {
  const res = await fetch('/api/v1/supply/preview/sellers.json', { credentials: 'same-origin' });
  if (!res.ok) throw new Error('sellers.json preview failed');
  const text = await res.text();
  try {
    return JSON.stringify(JSON.parse(text), null, 2);
  } catch {
    return text;
  }
}

/**
 * Fetch ads.txt preview text.
 */
export async function fetchAdsTxtPreview(): Promise<string> {
  const res = await fetch('/api/v1/supply/preview/ads.txt', { credentials: 'same-origin' });
  if (!res.ok) throw new Error('ads.txt preview failed');
  return res.text();
}

/**
 * Fetch the configured supply export path.
 */
export async function fetchSupplyExportPath(): Promise<string> {
  const { data } = await api('/api/v1/supply/export-path');
  const payload = data as { path?: string } | null | undefined;
  return payload?.path ?? '';
}
