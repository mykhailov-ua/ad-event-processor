import { apiFetch, apiJson } from './client.js';
import type {
  AdsTxtEntry,
  AdsTxtWriteRequest,
  Seller,
  SellerWriteRequest,
  SupplyExportPath,
  SupplyValidation,
} from './types.js';

/** Server-rendered preview paths (do not invent alternate URLs). */
export const SUPPLY_PREVIEW_ADS_TXT_PATH = '/api/v1/supply/preview/ads.txt';
export const SUPPLY_PREVIEW_SELLERS_JSON_PATH = '/api/v1/supply/preview/sellers.json';

export async function listSupplySellers(signal?: AbortSignal): Promise<Seller[]> {
  return apiJson<Seller[]>('/api/v1/supply/sellers', { signal });
}

export async function createSupplySeller(
  body: SellerWriteRequest,
  signal?: AbortSignal,
): Promise<Seller> {
  return apiJson<Seller>('/api/v1/supply/sellers', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function updateSupplySeller(
  id: number,
  body: SellerWriteRequest,
  signal?: AbortSignal,
): Promise<Seller> {
  return apiJson<Seller>(`/api/v1/supply/sellers/${encodeURIComponent(String(id))}`, {
    method: 'PUT',
    body: JSON.stringify(body),
    signal,
  });
}

export async function deleteSupplySeller(id: number, signal?: AbortSignal): Promise<void> {
  const response = await apiFetch(`/api/v1/supply/sellers/${encodeURIComponent(String(id))}`, {
    method: 'DELETE',
    signal,
  });
  if (!response.ok && response.status !== 204) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}

export async function listSupplyAdsTxt(signal?: AbortSignal): Promise<AdsTxtEntry[]> {
  return apiJson<AdsTxtEntry[]>('/api/v1/supply/ads-txt', { signal });
}

export async function createSupplyAdsTxt(
  body: AdsTxtWriteRequest,
  signal?: AbortSignal,
): Promise<AdsTxtEntry> {
  return apiJson<AdsTxtEntry>('/api/v1/supply/ads-txt', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function updateSupplyAdsTxt(
  id: number,
  body: AdsTxtWriteRequest,
  signal?: AbortSignal,
): Promise<AdsTxtEntry> {
  return apiJson<AdsTxtEntry>(`/api/v1/supply/ads-txt/${encodeURIComponent(String(id))}`, {
    method: 'PUT',
    body: JSON.stringify(body),
    signal,
  });
}

export async function deleteSupplyAdsTxt(id: number, signal?: AbortSignal): Promise<void> {
  const response = await apiFetch(`/api/v1/supply/ads-txt/${encodeURIComponent(String(id))}`, {
    method: 'DELETE',
    signal,
  });
  if (!response.ok && response.status !== 204) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}

export async function getSupplyExportPath(signal?: AbortSignal): Promise<SupplyExportPath> {
  return apiJson<SupplyExportPath>('/api/v1/supply/export-path', { signal });
}

export async function getSupplyValidation(signal?: AbortSignal): Promise<SupplyValidation> {
  return apiJson<SupplyValidation>('/api/v1/supply/validation', { signal });
}
