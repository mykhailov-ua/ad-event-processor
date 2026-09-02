import type { MockResult } from './handler_types.ts';

function json(status: number, body: unknown): MockResult {
  return { status, body, contentType: 'application/json' };
}

const DEV_DOCTOR_CHECKS = [
  {
    id: 'postgres',
    status: 'pass',
    message: 'Postgres reachable',
    hint: 'Dev mock',
    latency_ms: 3,
  },
  {
    id: 'redis',
    status: 'pass',
    message: 'Redis shards healthy',
    hint: 'Dev mock',
    latency_ms: 2,
  },
  {
    id: 'clickhouse',
    status: 'pass',
    message: 'ClickHouse lag within SLA',
    hint: 'Dev mock',
    latency_ms: 8,
  },
] as const;

export function devMockDoctorSummary() {
  return {
    overall: 'ok',
    checks: [...DEV_DOCTOR_CHECKS],
    tracking_domain: 'track.dev.local',
    rtb_mode: 'shadow',
    rtb_enabled: false,
    click_url_template: 'https://track.dev.local/click?cid={campaign_id}',
  };
}

export function devMockStackHealthSnapshot() {
  return {
    status: 'ok' as const,
    clickhouse_lag_seconds: 1.2,
    outbox_oldest_pending_seconds: 0.4,
    redis_shard_reachable: true,
    redis_shards_reachable: 4,
    redis_shards_total: 4,
    license_state: 'ACTIVE',
    cost_sync_last_success_seconds: 120,
    automation_worker_last_tick_seconds: 15,
  };
}

export function devMockDashboardSummary() {
  const now = new Date().toISOString();
  return {
    generated_at: now,
    generated_at_display: now,
    services: [
      { id: 'tracker', name: 'Tracker', status: 'ok', detail: 'Dev mock' },
      { id: 'control', name: 'Control plane', status: 'ok', detail: 'Dev mock' },
      { id: 'processor', name: 'Processor', status: 'ok', detail: 'Dev mock' },
    ],
    rps_estimate: 1240.5,
    outbox_pending: 0,
    drift_micro_max: 0,
    drift_alert: false,
    emergency_breaker: '',
  };
}

export function devMockOpsHomeSnapshot(): MockResult {
  return json(200, {
    doctor: devMockDoctorSummary(),
    stackHealth: devMockStackHealthSnapshot(),
    dashboardSummary: devMockDashboardSummary(),
  });
}

export function devMockIncidentSnapshot(): MockResult {
  return json(200, {
    emergency_breaker: '',
    shards: [
      {
        shard_id: 0,
        ping_ok: true,
        ping_latency_ms: 1.4,
        config_version_synced: true,
      },
      {
        shard_id: 1,
        ping_ok: true,
        ping_latency_ms: 1.8,
        config_version_synced: true,
      },
    ],
    affected_campaigns: [],
    partial: false,
    stale_dashboard: false,
  });
}

export function devMockOpsShardsResponse(): MockResult {
  return json(200, {
    emergency_breaker: '',
    shards: [
      {
        shard_id: 0,
        ping_ok: true,
        ping_latency_ms: 1.2,
        config_version: 42,
        config_version_lag: 0,
        config_version_synced: true,
      },
      {
        shard_id: 1,
        ping_ok: true,
        ping_latency_ms: 1.5,
        config_version: 42,
        config_version_lag: 0,
        config_version_synced: true,
      },
    ],
  });
}

export function devMockOpsObject(): MockResult {
  return json(200, {});
}

export function devMockOpsList(limit = 50, offset = 0): MockResult {
  return json(200, { items: [], total: 0, limit, offset });
}
