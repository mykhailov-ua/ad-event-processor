import { apiJson } from './client.js';
import type { Customer, CustomerListQuery, CustomerListResponse } from './types.js';

export function buildCustomersListPath(params: CustomerListQuery = {}): string {
  const search = new URLSearchParams();

  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }
  if (params.sort) {
    search.set('sort', params.sort);
  }
  if (params.order) {
    search.set('order', params.order);
  }

  const query = search.toString();
  return query ? `/api/v1/customers?${query}` : '/api/v1/customers';
}

export async function listCustomers(
  params: CustomerListQuery = {},
  signal?: AbortSignal,
): Promise<CustomerListResponse> {
  return apiJson<CustomerListResponse>(buildCustomersListPath(params), { signal });
}

export async function getCustomer(id: string, signal?: AbortSignal): Promise<Customer> {
  return apiJson<Customer>(`/api/v1/customers/${encodeURIComponent(id)}`, { signal });
}

export type PatchCustomerCostCenterRequest = {
  cost_center: string;
};

export async function patchCustomerCostCenter(
  id: string,
  body: PatchCustomerCostCenterRequest,
  signal?: AbortSignal,
): Promise<Customer> {
  return apiJson<Customer>(`/api/v1/customers/${encodeURIComponent(id)}/cost-center`, {
    method: 'PATCH',
    body: JSON.stringify(body),
    signal,
  });
}
