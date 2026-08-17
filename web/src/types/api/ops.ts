/** Go: doctor.Check / DoctorResponseDTO */

export type DoctorCheck = {
  id: string;
  status: string;
  message: string;
  hint?: string;
  latency_ms?: number;
};

export type OpsDoctorSummary = {
  overall: string;
  checks: DoctorCheck[];
  click_url_template: string;
  tracking_domain: string;
  rtb_mode?: string;
  rtb_enabled: boolean;
};

export type OutboxHealthSummary = {
  pending: number;
  oldest_pending_seconds?: number;
  last_processed_event_id?: number;
};

export type ShardHealthStatus = {
  shard_id: number;
  ping_ok: boolean;
  ping_error?: string;
  ping_latency_ms?: number;
  config_version?: number | null;
  config_version_lag?: number;
  config_version_synced?: boolean;
};

export type IncidentSnapshot = {
  emergency_breaker: string;
  shards: ShardHealthStatus[];
  outbox?: OutboxHealthSummary;
  stream_lag?: unknown[];
  breaker_states?: Record<string, string>;
  partial?: boolean;
  errors?: Array<{ source: string; code: string }>;
  stale_dashboard?: boolean;
  affected_campaigns?: Array<{ campaign_id: string; name?: string }>;
};

/** GET /ops/dashboard/summary */
export type DashboardSummary = {
  generated_at?: string;
  services?: Array<{ id?: string; name?: string; status?: string; detail?: string }>;
  drift_micro_max?: number;
  drift_alert?: boolean;
  rps_estimate?: number;
  outbox_pending?: number;
  emergency_breaker?: string;
};
