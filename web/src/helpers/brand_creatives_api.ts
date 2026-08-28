import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import type { components } from '../types/generated/openapi.js';

export type BrandDTO = components['schemas']['Brand'];
export type BrandCreativeDTO = components['schemas']['BrandCreative'];
export type UpsertBrandCreativeBody = components['schemas']['UpsertBrandCreativeRequest'];
export type UpdateBrandCreativeBody = components['schemas']['UpdateBrandCreativeRequest'];

/**
 * List brands for a customer.
 */
export async function fetchBrands(customerId: string): Promise<BrandDTO[]> {
  const { data } = await api<BrandDTO[]>(
    `/api/v1/brands?customer_id=${encodeURIComponent(customerId)}`
  );
  return Array.isArray(data) ? data : [];
}

/**
 * Create a brand under a customer.
 */
export async function createBrand(customerId: string, name: string): Promise<string> {
  const res = await apiConfirmed<{ id?: string }>('/api/v1/brands', {
    method: 'POST',
    body: JSON.stringify({ customer_id: customerId, name }),
  });
  return res.data?.id ?? '';
}

/**
 * List creatives for a brand.
 */
export async function fetchBrandCreatives(brandId: string): Promise<BrandCreativeDTO[]> {
  const { data } = await api<BrandCreativeDTO[]>(
    `/api/v1/brands/${encodeURIComponent(brandId)}/creatives`
  );
  return Array.isArray(data) ? data : [];
}

/**
 * Create a brand creative rotation entry.
 */
export async function createBrandCreative(
  brandId: string,
  body: UpsertBrandCreativeBody
): Promise<string> {
  const res = await apiConfirmed<{ id?: string }>(
    `/api/v1/brands/${encodeURIComponent(brandId)}/creatives`,
    {
      method: 'POST',
      body: JSON.stringify(body),
    }
  );
  return res.data?.id ?? '';
}

/**
 * Patch an existing brand creative.
 */
export async function updateBrandCreative(
  creativeId: string,
  body: UpdateBrandCreativeBody
): Promise<void> {
  await apiConfirmed(`/api/v1/brand-creatives/${encodeURIComponent(creativeId)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  });
}

/**
 * Delete a brand creative.
 */
export async function deleteBrandCreative(creativeId: string): Promise<void> {
  await apiConfirmed(`/api/v1/brand-creatives/${encodeURIComponent(creativeId)}`, {
    method: 'DELETE',
  });
}
