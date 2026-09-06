import { listCustomers } from '@/api/customers_api';
import type { CustomerListQuery, CustomerListResponse } from '@/api/types';

/** Canonical combobox picker query shared by campaigns, dashboard, and click log. */
export const CUSTOMERS_COMBOBOX_LIST_QUERY: CustomerListQuery = {
  limit: 100,
  offset: 0,
  sort: 'name',
  order: 'asc',
};

let cachedResponse: CustomerListResponse | undefined;
let inflightResponse: Promise<CustomerListResponse> | undefined;

/**
 * Fetches the customer combobox snapshot once per browser session; parallel mounts share inflight.
 * Paginated customers directory uses `listCustomers` directly with page query params.
 */
export function fetchCustomersComboboxCached(signal?: AbortSignal): Promise<CustomerListResponse> {
  if (cachedResponse) {
    return Promise.resolve(cachedResponse);
  }

  if (inflightResponse) {
    return inflightResponse;
  }

  inflightResponse = listCustomers(CUSTOMERS_COMBOBOX_LIST_QUERY, signal)
    .then((response) => {
      cachedResponse = response;
      return response;
    })
    .finally(() => {
      inflightResponse = undefined;
    });

  return inflightResponse;
}

export function invalidateCustomersComboboxCache(): void {
  cachedResponse = undefined;
  inflightResponse = undefined;
}
