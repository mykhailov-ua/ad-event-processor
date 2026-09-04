import { ApiError } from '@/api/api_error';

/** list-facets / metrics-totals are optional; stale control builds may return 403/404. */
export function isCampaignListAuxEndpointUnavailable(err: unknown): boolean {
  if (!(err instanceof ApiError)) {
    return false;
  }
  return err.status === 403 || err.status === 404 || err.status === 501;
}
