import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import type { components } from '../types/generated/openapi.js';

export type Seller = components['schemas']['Seller'];
export type SellerWriteRequest = components['schemas']['SellerWriteRequest'];
export type AdsTxtEntry = components['schemas']['AdsTxtEntry'];
export type AdsTxtWriteRequest = components['schemas']['AdsTxtWriteRequest'];
export type SupplyValidation = components['schemas']['SupplyValidation'];
export type SupplyExportPath = components['schemas']['SupplyExportPath'];

/**
 * List sellers.json rows stored in Postgres.
 */
export async function fetchSellers(): Promise<Seller[]> {
  const { data } = await api('/api/v1/supply/sellers');
  return (data as Seller[] | null | undefined) ?? [];
}

/**
 * Create a sellers.json row.
 */
export async function createSeller(body: SellerWriteRequest): Promise<Seller> {
  const res = await apiConfirmed('/api/v1/supply/sellers', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  return res.data as Seller;
}

/**
 * Update a sellers.json row by numeric id.
 */
export async function updateSeller(id: number, body: SellerWriteRequest): Promise<Seller> {
  const res = await apiConfirmed(`/api/v1/supply/sellers/${id}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
  return res.data as Seller;
}

/**
 * Delete a sellers.json row.
 */
export async function deleteSeller(id: number): Promise<void> {
  await apiConfirmed(`/api/v1/supply/sellers/${id}`, { method: 'DELETE' });
}

/**
 * List ads.txt rows stored in Postgres.
 */
export async function fetchAdsTxtEntries(): Promise<AdsTxtEntry[]> {
  const { data } = await api('/api/v1/supply/ads-txt');
  return (data as AdsTxtEntry[] | null | undefined) ?? [];
}

/**
 * Create an ads.txt row.
 */
export async function createAdsTxtEntry(body: AdsTxtWriteRequest): Promise<AdsTxtEntry> {
  const res = await apiConfirmed('/api/v1/supply/ads-txt', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  return res.data as AdsTxtEntry;
}

/**
 * Update an ads.txt row.
 */
export async function updateAdsTxtEntry(
  id: number,
  body: AdsTxtWriteRequest
): Promise<AdsTxtEntry> {
  const res = await apiConfirmed(`/api/v1/supply/ads-txt/${id}`, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
  return res.data as AdsTxtEntry;
}

/**
 * Delete an ads.txt row.
 */
export async function deleteAdsTxtEntry(id: number): Promise<void> {
  await apiConfirmed(`/api/v1/supply/ads-txt/${id}`, { method: 'DELETE' });
}

/**
 * Fetch rendered sellers.json preview text.
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
 * Fetch rendered ads.txt preview text.
 */
export async function fetchAdsTxtPreview(): Promise<string> {
  const res = await fetch('/api/v1/supply/preview/ads.txt', { credentials: 'same-origin' });
  if (!res.ok) throw new Error('ads.txt preview failed');
  return res.text();
}

/**
 * Read nginx export path for supply files.
 */
export async function fetchSupplyExportPath(): Promise<string> {
  const { data } = await api('/api/v1/supply/export-path');
  const payload = data as SupplyExportPath | null | undefined;
  return payload?.path ?? '';
}

/**
 * Validate sellers.json and ads.txt export artifacts.
 */
export async function fetchSupplyValidation(): Promise<SupplyValidation> {
  const { data } = await api('/api/v1/supply/validation');
  return data as SupplyValidation;
}
