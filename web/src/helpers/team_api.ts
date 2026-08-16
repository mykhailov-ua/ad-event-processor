import { api } from './api_client.js';
import type { TeamOverviewDTO } from '../types/api/team.js';

/**
 * Load team overview for the bound or selected customer.
 */
export async function fetchTeamOverview(customerId?: string): Promise<TeamOverviewDTO> {
  const params = customerId ? `?customer_id=${encodeURIComponent(customerId)}` : '';
  const res = await api<TeamOverviewDTO>(`/api/v1/team/overview${params}`);
  return res.data;
}
