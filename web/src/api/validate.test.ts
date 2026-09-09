import assert from 'node:assert/strict';
import test from 'node:test';

import { ApiError } from '@/api/api_error';
import {
  isCampaignWizardCommitResult,
  isCampaignWizardSession,
  parseAuditLogRow,
  parseCampaign,
  parseCampaignBulkActionResponse,
  parseCampaignFlowValidateResponse,
  parseCampaignListMetricsTotalsResponse,
  parseCampaignListResponse,
  parseFlow,
  parseFlowPath,
  parseFlowPathList,
  parseFraudCatalogReportResponse,
  parseFraudBreakdownReportResponse,
  parseWireSignalBreakdownReportResponse,
  parseCampaignWizardCommitResult,
  parseCampaignWizardSession,
  parseLanderListResponse,
  parsePostbackDryRunResult,
} from '@/api/validate.ts';

test('parseAuditLogRow rejects non-object payload', () => {
  assert.throws(
    () => parseAuditLogRow(null),
    (err: unknown) => {
      return err instanceof ApiError && err.code === 'INVALID_RESPONSE';
    }
  );
});

test('parseCampaign rejects missing customer_id', () => {
  assert.throws(
    () =>
      parseCampaign({
        id: 'abe62900-7466-5a77-8dac-2cf17fd1dd84',
        name: 'c1',
        status: 'draft',
      }),
    (err: unknown) => err instanceof ApiError && err.code === 'INVALID_RESPONSE'
  );
});

test('parseCampaignListMetricsTotalsResponse rejects missing totals object', () => {
  assert.throws(
    () =>
      parseCampaignListMetricsTotalsResponse({
        campaign_count: 1,
        flow_count: 1,
        margin_breach_count: 0,
        from: '2026-01-01T00:00:00.000Z',
        to: '2026-01-02T00:00:00.000Z',
        stale: false,
      }),
    (err: unknown) => err instanceof ApiError && err.code === 'INVALID_RESPONSE'
  );
});

test('parseFraudBreakdownReportResponse rejects non-object row', () => {
  assert.throws(
    () => parseFraudBreakdownReportResponse({ rows: [null] }),
    (err: unknown) => err instanceof ApiError && err.code === 'INVALID_RESPONSE'
  );
});

test('parseWireSignalBreakdownReportResponse accepts wire rows', () => {
  const parsed = parseWireSignalBreakdownReportResponse({
    rows: [{ fraud_reason: 'bot', signals_degraded: true }],
    freshness: {
      as_of: '2026-01-01T00:00:00.000Z',
      consistency: 'eventual',
      stale: false,
    },
  });
  assert.equal(parsed.rows.length, 1);
  assert.equal(parsed.rows[0]?.signals_degraded, true);
});

test('parseFraudCatalogReportResponse rejects missing freshness', () => {
  assert.throws(
    () =>
      parseFraudCatalogReportResponse('signal-effectiveness', {
        rows: [{ signal_code: 'ja3' }],
      }),
    (err: unknown) => err instanceof ApiError && err.code === 'INVALID_RESPONSE'
  );
});

test('parseFraudCatalogReportResponse validates filter-reject rows', () => {
  const parsed = parseFraudCatalogReportResponse('filter-rejects', {
    rows: [{ reject_kind: 'geo', reject_count: 12 }],
    freshness: {
      as_of: '2026-01-01T00:00:00.000Z',
      consistency: 'eventual',
      stale: false,
    },
  });
  assert.equal(parsed.rows[0]?.reject_kind, 'geo');
});

test('parseFlow validates string paths without parsing JSON', () => {
  const parsed = parseFlow({
    id: '00000000-0000-4000-8000-000000000001',
    name: 'Main',
    paths: 'legacy-json-string',
    created_at: '2026-01-01T00:00:00.000Z',
  });
  assert.equal(parsed.paths, 'legacy-json-string');
});

test('parseFlowPathList parses path array from wire', () => {
  const parsed = parseFlowPathList([
    {
      weight: 100,
      landers: [{ lander_id: '00000000-0000-4000-8000-000000000003', weight: 100 }],
      offers: [{ offer_id: '00000000-0000-4000-8000-000000000004', weight: 100 }],
    },
  ]);
  assert.equal(parsed.length, 1);
  assert.equal(parsed[0]?.weight, 100);
});

test('parseFlowPath validates optional filters', () => {
  const parsed = parseFlowPath({
    weight: 100,
    landers: [{ lander_id: '00000000-0000-4000-8000-000000000003', weight: 100 }],
    offers: [{ offer_id: '00000000-0000-4000-8000-000000000004', weight: 100 }],
    filters: { countries: ['US'], devices: ['mobile'] },
  });
  assert.deepEqual(parsed.filters?.countries, ['US']);
});

test('parseFlowPath rejects invalid filters.devices', () => {
  assert.throws(
    () =>
      parseFlowPath({
        weight: 100,
        landers: [{ lander_id: '00000000-0000-4000-8000-000000000003', weight: 100 }],
        offers: [{ offer_id: '00000000-0000-4000-8000-000000000004', weight: 100 }],
        filters: { devices: ['mobile', 1] },
      }),
    (err: unknown) => err instanceof ApiError && err.code === 'INVALID_RESPONSE'
  );
});

test('parseCampaignListResponse rejects missing pagination', () => {
  assert.throws(
    () =>
      parseCampaignListResponse({
        items: [
          {
            id: 'abe62900-7466-5a77-8dac-2cf17fd1dd84',
            name: 'c1',
            status: 'draft',
            customer_id: '00000000-0000-4000-8000-000000000002',
          },
        ],
        total: 1,
      }),
    (err: unknown) => err instanceof ApiError && err.code === 'INVALID_RESPONSE'
  );
});

test('parseCampaignBulkActionResponse accepts valid results', () => {
  const parsed = parseCampaignBulkActionResponse({
    results: [{ id: 'abe62900-7466-5a77-8dac-2cf17fd1dd84', ok: true }],
  });
  assert.equal(parsed.results.length, 1);
  assert.equal(parsed.results[0]?.ok, true);
});

test('parseCampaignFlowValidateResponse rejects missing valid flag', () => {
  assert.throws(
    () => parseCampaignFlowValidateResponse({ path_errors: [] }),
    (err: unknown) => {
      return err instanceof ApiError && err.code === 'INVALID_RESPONSE';
    }
  );
});

test('parseLanderListResponse rejects missing pagination', () => {
  assert.throws(
    () =>
      parseLanderListResponse({
        items: [],
        hosting_counts: { total: 0, external: 0, hosted: 0, unconfigured: 0 },
      }),
    (err: unknown) => err instanceof ApiError && err.code === 'INVALID_RESPONSE'
  );
});

test('parseCampaignWizardSession rejects commit-shaped payload', () => {
  assert.throws(
    () =>
      parseCampaignWizardSession({
        campaign: { id: 'abe62900-7466-5a77-8dac-2cf17fd1dd84', name: 'c1' },
      }),
    (err: unknown) => err instanceof ApiError && err.code === 'INVALID_RESPONSE'
  );
});

test('parseCampaignWizardCommitResult rejects session-shaped payload', () => {
  assert.throws(
    () =>
      parseCampaignWizardCommitResult({
        session_id: 'abe62900-7466-5a77-8dac-2cf17fd1dd84',
        current_step: 'traffic_source',
        completed_steps: [],
        ready_to_commit: false,
      }),
    (err: unknown) => err instanceof ApiError && err.code === 'INVALID_RESPONSE'
  );
});

test('isCampaignWizardSession and isCampaignWizardCommitResult discriminate union', () => {
  const session = {
    session_id: 'abe62900-7466-5a77-8dac-2cf17fd1dd84',
    current_step: 'traffic_source',
    completed_steps: [] as string[],
    ready_to_commit: false,
  };
  const commit = {
    campaign: { id: 'abe62900-7466-5a77-8dac-2cf17fd1dd84', name: 'c1' },
  };
  assert.equal(isCampaignWizardSession(session), true);
  assert.equal(isCampaignWizardCommitResult(session), false);
  assert.equal(isCampaignWizardCommitResult(commit), true);
  assert.equal(isCampaignWizardSession(commit), false);
});

test('parsePostbackDryRunResult rejects missing ok flag', () => {
  assert.throws(
    () => parsePostbackDryRunResult({ provider: 'meta', test_event: true }),
    (err: unknown) => err instanceof ApiError && err.code === 'INVALID_RESPONSE'
  );
});

test('parsePostbackDryRunResult accepts 422-shaped dry-run body', () => {
  const parsed = parsePostbackDryRunResult({
    ok: false,
    provider: 'meta',
    test_event: true,
    error: 'timeout',
    http_status: 504,
  });
  assert.equal(parsed.ok, false);
  assert.equal(parsed.provider, 'meta');
});

test('parseLanderListResponse accepts paginated lander page', () => {
  const parsed = parseLanderListResponse({
    items: [
      {
        id: '00000000-0000-4000-8000-000000000001',
        name: 'Hosted LP',
        created_at: '2026-01-01T00:00:00.000Z',
      },
    ],
    total: 1,
    limit: 25,
    offset: 0,
    hosting_counts: { total: 3, external: 1, hosted: 1, unconfigured: 1 },
  });
  assert.equal(parsed.items.length, 1);
  assert.equal(parsed.total, 1);
  assert.equal(parsed.hosting_counts.hosted, 1);
});
