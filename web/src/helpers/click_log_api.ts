import { api } from './api_client.js';
import type { DataFreshness } from '../types/report.js';

/** One row in the click log timeline or browse list. */
export type ClickLogEvent = {
  event_type: string;
  click_id: string;
  campaign_id: string;
  placement_id?: string;
  created_at: string;
  attributed_cost_micro?: number;
  cost_source?: string;
  revenue_micro?: number;
  inbound_status?: string;
  goal_name?: string;
  sub1?: string;
  country?: string;
};

/** Outbound CAPI/postback dispatch row for a click. */
export type ClickLogPostback = {
  status: string;
  error_message?: string;
  created_at: string;
};

export type ClickLogResponse = {
  events: ClickLogEvent[];
  postbacks?: ClickLogPostback[];
  freshness?: DataFreshness;
  next_cursor?: string;
};

export type ClickLogQuery = {
  customerId: string;
  from: string;
  to: string;
  clickId?: string;
  campaignId?: string;
  cursor?: string;
};

/**
 * Fetch click log events from ClickHouse-backed report API.
 */
export async function fetchClickLog(query: ClickLogQuery): Promise<ClickLogResponse> {
  const params = new URLSearchParams({
    customer_id: query.customerId,
    from: query.from,
    to: query.to,
    limit: '50',
  });
  if (query.clickId) params.set('click_id', query.clickId);
  if (query.campaignId) params.set('campaign_id', query.campaignId);
  if (query.cursor) params.set('cursor', query.cursor);
  const { data } = await api(`/api/v1/reports/click-log?${params.toString()}`);
  const body = (data as ClickLogResponse | null) ?? { events: [] };
  return {
    events: body.events ?? [],
    postbacks: body.postbacks ?? [],
    freshness: body.freshness,
    next_cursor: body.next_cursor,
  };
}
