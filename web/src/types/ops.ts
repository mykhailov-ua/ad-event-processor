import type { components } from './generated/openapi.js';

export type DoctorCheck = components['schemas']['DoctorCheck'];

export type OpsDoctorSummary = components['schemas']['DoctorSummary'] & {
  overall: string;
};

export type DashboardSummary = components['schemas']['DashboardSummary'];

export type OutboxHealthSummary = components['schemas']['OutboxHealthSummary'];

export type ShardHealthStatus = components['schemas']['ShardHealthStatus'];

export type IncidentSnapshot = components['schemas']['IncidentSnapshot'] & {
  affected_campaigns?: Array<{ campaign_id: string; name?: string }>;
};

export type DashboardMetrics = components['schemas']['DashboardMetrics'];
