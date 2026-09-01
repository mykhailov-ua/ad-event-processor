import { apiJson } from './client.js';
import type {
  Brand,
  BrandCreative,
  BrandsListQuery,
  CreateBrandRequest,
  UpdateBrandCreativeRequest,
} from './types.js';

export async function listBrands(
  params: BrandsListQuery,
  signal?: AbortSignal,
): Promise<Brand[]> {
  const search = new URLSearchParams();
  search.set('customer_id', params.customer_id);
  return apiJson<Brand[]>(`/api/v1/brands?${search.toString()}`, { signal });
}

export async function createBrand(
  body: CreateBrandRequest,
  signal?: AbortSignal,
): Promise<Brand> {
  return apiJson<Brand>('/api/v1/brands', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function listBrandCreatives(
  brandId: string,
  signal?: AbortSignal,
): Promise<BrandCreative[]> {
  return apiJson<BrandCreative[]>(`/api/v1/brands/${encodeURIComponent(brandId)}/creatives`, {
    signal,
  });
}

export async function createBrandCreative(
  brandId: string,
  body: UpdateBrandCreativeRequest,
  signal?: AbortSignal,
): Promise<BrandCreative> {
  return apiJson<BrandCreative>(`/api/v1/brands/${encodeURIComponent(brandId)}/creatives`, {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function patchBrandCreative(
  creativeId: string,
  body: UpdateBrandCreativeRequest,
  signal?: AbortSignal,
): Promise<BrandCreative> {
  return apiJson<BrandCreative>(`/api/v1/brand-creatives/${encodeURIComponent(creativeId)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
    signal,
  });
}

export async function deleteBrandCreative(creativeId: string, signal?: AbortSignal): Promise<void> {
  await apiJson<void>(`/api/v1/brand-creatives/${encodeURIComponent(creativeId)}`, {
    method: 'DELETE',
    signal,
  });
}
