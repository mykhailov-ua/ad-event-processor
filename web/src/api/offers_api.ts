import { apiFetch, apiJson, apiJsonArray, parseApiError } from './client.js';
import type { CreateOfferRequest, Offer, UpdateOfferRequest } from './types.js';

export async function listOffers(signal?: AbortSignal): Promise<Offer[]> {
  return apiJsonArray<Offer>('/api/v1/offers', { signal });
}

export async function getOffer(offerId: string, signal?: AbortSignal): Promise<Offer> {
  return apiJson<Offer>(`/api/v1/offers/${encodeURIComponent(offerId)}`, { signal });
}

export async function createOffer(body: CreateOfferRequest, signal?: AbortSignal): Promise<Offer> {
  return apiJson<Offer>('/api/v1/offers', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function updateOffer(
  offerId: string,
  body: UpdateOfferRequest,
  signal?: AbortSignal
): Promise<Offer> {
  return apiJson<Offer>(`/api/v1/offers/${encodeURIComponent(offerId)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
    signal,
  });
}

export async function deleteOffer(offerId: string, signal?: AbortSignal): Promise<void> {
  const response = await apiFetch(`/api/v1/offers/${encodeURIComponent(offerId)}`, {
    method: 'DELETE',
    signal,
  });
  if (!response.ok) {
    throw await parseApiError(response);
  }
}
