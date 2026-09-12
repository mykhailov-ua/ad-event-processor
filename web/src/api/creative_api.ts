import { apiJsonValidated } from './client.js';
import { parseLanderListResponse, parseOfferList } from './validate.js';
import type { Lander, LanderListQuery, Offer } from './types.js';

export async function listLanders(
  query: LanderListQuery = {},
  signal?: AbortSignal
): Promise<Lander[]> {
  const search = new URLSearchParams();
  if (query.q) {
    search.set('q', query.q);
  }
  if (query.hosting) {
    search.set('hosting', query.hosting);
  }
  if (query.limit != null) {
    search.set('limit', String(query.limit));
  }
  if (query.offset != null) {
    search.set('offset', String(query.offset));
  }
  const qs = search.toString();
  const path = qs ? `/api/v1/landers?${qs}` : '/api/v1/landers';
  const response = await apiJsonValidated(path, { signal }, parseLanderListResponse);
  return response.items;
}

export async function listOffers(signal?: AbortSignal): Promise<Offer[]> {
  return apiJsonValidated('/api/v1/offers', { signal }, parseOfferList);
}
