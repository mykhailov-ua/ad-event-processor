import { api } from './api_client.js';
import type { PublisherDashboard, PublisherStatementList } from '../types/publisher.js';

export async function fetchPublisherDashboard(query = ''): Promise<PublisherDashboard> {
  const q = query ? (query.startsWith('?') ? query : `?${query}`) : '';
  const { data } = await api<PublisherDashboard>(`/api/v1/publisher/dashboard${q}`);
  return data as PublisherDashboard;
}

export async function fetchPublisherStatements(query = ''): Promise<PublisherStatementList> {
  const q = query ? (query.startsWith('?') ? query : `?${query}`) : '';
  const { data } = await api<PublisherStatementList>(`/api/v1/publisher/statements${q}`);
  return data as PublisherStatementList;
}
