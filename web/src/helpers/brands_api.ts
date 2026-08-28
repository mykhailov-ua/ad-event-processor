import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type Brand = {
  id?: string;
  customer_id?: string;
  name?: string;
  updated_at?: string;
  freq_limit?: number;
  freq_window?: number;
};

export type BrandCreative = {
  id?: string;
  brand_id?: string;
  name?: string;
  landing_url?: string;
  weight?: number;
  status?: string;
  updated_at?: string;
};

export function buildBrandsListUrl(customerId: string): string {
  const qs = new URLSearchParams({ customer_id: customerId });
  return `/api/v1/brands?${qs.toString()}`;
}

export async function fetchBrands(customerId: string, signal?: AbortSignal): Promise<Brand[]> {
  const result = await api<Brand[]>(buildBrandsListUrl(customerId), { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function createBrand(body: {
  customer_id: string;
  name: string;
}): Promise<string> {
  const result = await apiConfirmed<{ id?: string }>('/api/v1/brands', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('create brand failed');
  }
  return result.data?.id ?? '';
}

export async function fetchBrandCreatives(
  brandId: string,
  signal?: AbortSignal
): Promise<BrandCreative[]> {
  const result = await api<BrandCreative[]>(
    `/api/v1/brands/${encodeURIComponent(brandId)}/creatives`,
    { signal }
  );
  return Array.isArray(result.data) ? result.data : [];
}

export async function createBrandCreative(
  brandId: string,
  body: { name: string; landing_url: string; weight: number; status: string }
): Promise<void> {
  const result = await apiConfirmed(`/api/v1/brands/${encodeURIComponent(brandId)}/creatives`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('create creative failed');
  }
}
