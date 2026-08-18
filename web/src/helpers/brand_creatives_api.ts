import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export async function fetchBrands(customerId: string): Promise<unknown[]> {
  const { data } = await api(`/api/v1/brands?customer_id=${encodeURIComponent(customerId)}`);
  return (data as unknown[] | null | undefined) ?? [];
}

export async function createBrand(customerId: string, name: string): Promise<string> {
  const res = await apiConfirmed('/api/v1/brands', {
    method: 'POST',
    body: JSON.stringify({ customer_id: customerId, name }),
  });
  const payload = res.data as { id?: string } | null | undefined;
  return payload?.id ?? '';
}

export async function fetchBrandCreatives(brandId: string): Promise<unknown[]> {
  const { data } = await api(`/api/v1/brands/${encodeURIComponent(brandId)}/creatives`);
  return (data as unknown[] | null | undefined) ?? [];
}

export async function createBrandCreative(
  brandId: string,
  body: Record<string, unknown>
): Promise<string> {
  const res = await apiConfirmed(`/api/v1/brands/${encodeURIComponent(brandId)}/creatives`, {
    method: 'POST',
    body: JSON.stringify(body),
  });
  const payload = res.data as { id?: string } | null | undefined;
  return payload?.id ?? '';
}

export async function updateBrandCreative(
  creativeId: string,
  body: Record<string, unknown>
): Promise<void> {
  await apiConfirmed(`/api/v1/brand-creatives/${encodeURIComponent(creativeId)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  });
}

export async function deleteBrandCreative(creativeId: string): Promise<void> {
  await apiConfirmed(`/api/v1/brand-creatives/${encodeURIComponent(creativeId)}`, {
    method: 'DELETE',
  });
}
