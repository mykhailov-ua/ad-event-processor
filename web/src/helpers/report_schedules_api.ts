import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type ReportSchedule = {
  id?: string;
  customer_id?: string;
  report_key?: string;
  format?: string;
  cron_expr?: string;
  spec?: Record<string, unknown>;
  enabled?: boolean;
  next_run_at?: string;
  last_run_at?: string;
};

export type CreateReportScheduleRequest = {
  customer_id: string;
  report_key: string;
  format?: string;
  cron_expr: string;
  spec?: Record<string, unknown>;
  enabled?: boolean;
};

export type UpdateReportScheduleRequest = {
  report_key?: string;
  format?: string;
  cron_expr?: string;
  spec?: Record<string, unknown>;
  enabled?: boolean;
};

export function buildReportSchedulesListUrl(customerId: string): string {
  const qs = new URLSearchParams({ customer_id: customerId });
  return `/api/v1/report-schedules?${qs.toString()}`;
}

export async function fetchReportSchedules(
  customerId: string,
  signal?: AbortSignal
): Promise<ReportSchedule[]> {
  const result = await api<ReportSchedule[]>(buildReportSchedulesListUrl(customerId), { signal });
  return Array.isArray(result.data) ? result.data : [];
}

export async function createReportSchedule(
  body: CreateReportScheduleRequest
): Promise<ReportSchedule> {
  const result = await apiConfirmed<ReportSchedule>('/api/v1/report-schedules', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('create schedule failed');
  }
  return result.data ?? {};
}

export async function updateReportSchedule(
  id: string,
  body: UpdateReportScheduleRequest
): Promise<ReportSchedule> {
  const result = await apiConfirmed<ReportSchedule>(
    `/api/v1/report-schedules/${encodeURIComponent(id)}`,
    {
      method: 'PUT',
      body: JSON.stringify(body),
    }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('update schedule failed');
  }
  return result.data ?? {};
}

export async function deleteReportSchedule(id: string): Promise<void> {
  const result = await apiConfirmed(`/api/v1/report-schedules/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('delete schedule failed');
  }
}
