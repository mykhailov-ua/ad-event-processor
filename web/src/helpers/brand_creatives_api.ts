import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

/**
 * List brands for a customer.
 */
export async function fetchBrands(customerId: string): Promise<unknown[]> {
  const { data } = await api(`/api/v1/brands?customer_id=${encodeURIComponent(customerId)}`);
  return (data as unknown[] | null | undefined) ?? [];
}

/**
 * Create a brand and return its id.
 */
export async function createBrand(customerId: string, name: string): Promise<string> {
  const res = await apiConfirmed('/api/v1/brands', {
    method: 'POST',
    body: JSON.stringify({ customer_id: customerId, name }),
  });
  const payload = res.data as { id?: string } | null | undefined;
  return payload?.id ?? '';
}

/**
 * List creatives for a brand.
 */
export async function fetchBrandCreatives(brandId: string): Promise<unknown[]> {
  const { data } = await api(`/api/v1/brands/${encodeURIComponent(brandId)}/creatives`);
  return (data as unknown[] | null | undefined) ?? [];
}

/**
 * Create a brand creative and return its id.
 */
export async function createBrandCreative(
  brandId: string,
  body: Record<string, unknown>,
): Promise<string> {
  const res = await apiConfirmed(`/api/v1/brands/${encodeURIComponent(brandId)}/creatives`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
  const payload = res.data as { id?: string } | null | undefined;
  return payload?.id ?? '';
}

/**
 * Patch a brand creative.
 */
export async function updateBrandCreative(
  creativeId: string,
  body: Record<string, unknown>,
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
