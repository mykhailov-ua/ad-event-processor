import { apiFetch, apiJson, parseApiError } from './client.js';
import type {
  CreateReportScheduleRequest,
  ReportSchedule,
  ReportScheduleRunResponse,
  ReportSchedulesListQuery,
  UpdateReportScheduleRequest,
} from './types.js';

export async function listReportSchedules(
  params: ReportSchedulesListQuery,
  signal?: AbortSignal
): Promise<ReportSchedule[]> {
  const search = new URLSearchParams();
  if (params.customer_id) {
    search.set('customer_id', params.customer_id);
  }
  const query = search.toString();
  const path = query ? `/api/v1/report-schedules?${query}` : '/api/v1/report-schedules';
  return apiJson<ReportSchedule[]>(path, { signal });
}

export async function getReportSchedule(id: string, signal?: AbortSignal): Promise<ReportSchedule> {
  return apiJson<ReportSchedule>(`/api/v1/report-schedules/${encodeURIComponent(id)}`, { signal });
}

export async function createReportSchedule(
  body: CreateReportScheduleRequest,
  signal?: AbortSignal
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
  signal?: AbortSignal
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
    throw await parseApiError(response);
  }
}

export async function runReportScheduleNow(
  id: string,
  signal?: AbortSignal
): Promise<ReportScheduleRunResponse> {
  return apiJson<ReportScheduleRunResponse>(
    `/api/v1/report-schedules/${encodeURIComponent(id)}/run`,
    {
      method: 'POST',
      signal,
    }
  );
}
