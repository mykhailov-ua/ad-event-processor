import { apiJson } from './client.js';
import type {
  PublisherDashboard,
  PublisherStatementListResponse,
  PublisherStatementsQuery,
} from './types.js';

export async function getPublisherDashboard(
  params: PublisherStatementsQuery = {},
  signal?: AbortSignal,
): Promise<PublisherDashboard> {
  const search = new URLSearchParams();
  if (params.from) {
    search.set('from', params.from);
  }
  if (params.to) {
    search.set('to', params.to);
  }
  const query = search.toString();
  const path = query ? `/api/v1/publisher/dashboard?${query}` : '/api/v1/publisher/dashboard';
  return apiJson<PublisherDashboard>(path, { signal });
}

export async function listPublisherStatements(
  params: PublisherStatementsQuery = {},
  signal?: AbortSignal,
): Promise<PublisherStatementListResponse> {
  const search = new URLSearchParams();
  if (params.from) {
    search.set('from', params.from);
  }
  if (params.to) {
    search.set('to', params.to);
  }
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }
  const query = search.toString();
  const path = query ? `/api/v1/publisher/statements?${query}` : '/api/v1/publisher/statements';
  return apiJson<PublisherStatementListResponse>(path, { signal });
}
