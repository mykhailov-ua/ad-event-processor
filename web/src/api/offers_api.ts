import { apiJson } from './client.js';
import type { CreateOfferRequest, Offer } from './types.js';

export async function listOffers(signal?: AbortSignal): Promise<Offer[]> {
  return apiJson<Offer[]>('/api/v1/offers', { signal });
}

export async function createOffer(body: CreateOfferRequest, signal?: AbortSignal): Promise<Offer> {
  return apiJson<Offer>('/api/v1/offers', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}
