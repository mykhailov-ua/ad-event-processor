import type { MockResult } from './handler_types.ts';
import { devMockIso } from './fixture_helpers.ts';
import { SEED_DOMAIN_HOSTS, seedCatalogName } from './fixture_names.ts';
import { seedDeterministicUuid } from './seed_uuid.ts';
import { DEV_MOCK_CUSTOMERS } from './fixtures.ts';
import { devMockStore } from './store.ts';

function json(status: number, body: unknown): MockResult {
  return { status, body, contentType: 'application/json' };
}

const DEV_DOCTOR_CHECKS = [
  {
    id: 'postgres',
    status: 'pass',
    message: 'Postgres reachable',
    hint: 'Synthetic preview',
    latency_ms: 3,
  },
  {
    id: 'redis',
    status: 'pass',
    message: 'Redis shards healthy',
    hint: 'Synthetic preview',
    latency_ms: 2,
  },
  {
    id: 'clickhouse',
    status: 'pass',
    message: 'ClickHouse lag within SLA',
    hint: 'Synthetic preview',
    latency_ms: 8,
  },
] as const;

export function devMockDoctorSummary() {
  const trackingDomain = seedCatalogName(SEED_DOMAIN_HOSTS, 1);
  return {
    overall: 'ok',
    checks: [...DEV_DOCTOR_CHECKS],
    tracking_domain: trackingDomain,
    rtb_mode: 'shadow',
    rtb_enabled: false,
    click_url_template: `https://${trackingDomain}/click?cid={campaign_id}`,
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
      { id: 'tracker', name: 'Tracker', status: 'ok', detail: 'Within SLA' },
      { id: 'control', name: 'Control plane', status: 'ok', detail: 'Within SLA' },
      { id: 'processor', name: 'Processor', status: 'ok', detail: 'Within SLA' },
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

export function devMockOpsList(pathname: string, limit = 50, offset = 0): MockResult {
  const now = devMockIso(0);
  let items: unknown[] = [];

  if (pathname.includes('/ops/blacklist')) {
    items = Array.from({ length: 12 }, (_, index) => ({
      id: 2000 + index,
      ip: `203.0.113.${10 + index}`,
      reason: index % 2 === 0 ? 'fraud_score' : 'manual_block',
      created_at: devMockIso(index),
      created_at_display: devMockIso(index),
      expires_at: devMockIso(-index - 1),
      expires_at_display: devMockIso(-index - 1),
    }));
  } else if (pathname.includes('/ops/outbox')) {
    items = Array.from({ length: 15 }, (_, index) => ({
      id: 3000 + index,
      event_type: ['campaign_update', 'billing_sync', 'fraud_action'][index % 3],
      status: index % 4 === 0 ? 'pending' : 'applied',
      created_at: devMockIso(index % 10, index),
    }));
  } else if (pathname.includes('/ops/dlq/inbox')) {
    const campaigns = devMockStore().campaigns;
    items = Array.from({ length: 8 }, (_, index) => ({
      id: seedDeterministicUuid('ops_dlq', index + 1),
      source: 'processor',
      campaign_id: campaigns[index % campaigns.length]?.id,
      event_type: 'track',
      error: 'redis timeout',
      failed_at: devMockIso(index),
      failed_at_display: devMockIso(index),
      status: 'pending',
      retry_count: index % 3,
      shard_id: index % 4,
    }));
  } else if (pathname.includes('/ops/recon') || pathname.includes('/recon/runs')) {
    items = Array.from({ length: 6 }, (_, index) => ({
      id: seedDeterministicUuid('recon_run', index + 1),
      customer_id: DEV_MOCK_CUSTOMERS[index % DEV_MOCK_CUSTOMERS.length].id,
      status: index % 2 === 0 ? 'ok' : 'drift',
      diff_micro: index % 2 === 0 ? 0 : 12_500_000,
      created_at: now,
    }));
  }

  const page = items.slice(offset, offset + limit);
  return json(200, { items: page, total: items.length, limit, offset });
}
