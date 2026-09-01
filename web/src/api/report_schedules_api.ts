import { apiFetch, apiJson } from './client.js';
import type {
  CreateReportScheduleRequest,
  ReportSchedule,
  ReportSchedulesListQuery,
  UpdateReportScheduleRequest,
} from './types.js';

export async function listReportSchedules(
  params: ReportSchedulesListQuery,
  signal?: AbortSignal,
): Promise<ReportSchedule[]> {
  const search = new URLSearchParams();
  search.set('customer_id', params.customer_id);
  return apiJson<ReportSchedule[]>(`/api/v1/report-schedules?${search.toString()}`, { signal });
}

export async function getReportSchedule(id: string, signal?: AbortSignal): Promise<ReportSchedule> {
  return apiJson<ReportSchedule>(`/api/v1/report-schedules/${encodeURIComponent(id)}`, {
    signal,
  });
}

export async function createReportSchedule(
  body: CreateReportScheduleRequest,
  signal?: AbortSignal,
): Promise<ReportSchedule> {
  return apiJson<ReportSchedule>('/api/v1/report-schedules', {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function updateReportSchedule(
  id: string,
  body: UpdateReportScheduleRequest,
  signal?: AbortSignal,
): Promise<ReportSchedule> {
  return apiJson<ReportSchedule>(`/api/v1/report-schedules/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(body),
    signal,
  });
}

export async function deleteReportSchedule(id: string, signal?: AbortSignal): Promise<void> {
  const response = await apiFetch(`/api/v1/report-schedules/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    signal,
  });
  if (!response.ok && response.status !== 204) {
    throw new Error(response.statusText || `HTTP ${response.status}`);
  }
}
