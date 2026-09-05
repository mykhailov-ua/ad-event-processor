import { ApiError } from '@/api/api_error';

/** Stale control images route list-facets/metrics-totals to GET /campaigns/{id} and return invalid campaign id. */
export function isCampaignListAuxStaleControlRoute(err: unknown): boolean {
  if (!(err instanceof ApiError) || err.status !== 400) {
    return false;
  }
  const message = err.message.toLowerCase();
  return message.includes('invalid campaign id');
}

/** list-facets / metrics-totals are optional; stale control builds may return 403/404/501 or shadowed {id} 400. */
export function isCampaignListAuxEndpointUnavailable(err: unknown): boolean {
  if (isCampaignListAuxStaleControlRoute(err)) {
    return true;
  }
  if (!(err instanceof ApiError)) {
    return false;
  }
  return err.status === 403 || err.status === 404 || err.status === 501;
}
