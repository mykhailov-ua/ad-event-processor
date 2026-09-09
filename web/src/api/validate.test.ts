import assert from 'node:assert/strict';
import test from 'node:test';

import { ApiError } from '@/api/api_error';
import {
  parseAuditLogRow,
  parseCampaignBulkActionResponse,
  parseCampaignFlowValidateResponse,
  parseCampaignListResponse,
  parseLanderListResponse,
} from '@/api/validate.ts';

test('parseAuditLogRow rejects non-object payload', () => {
  assert.throws(
    () => parseAuditLogRow(null),
    (err: unknown) => {
      return err instanceof ApiError && err.code === 'INVALID_RESPONSE';
    }
  );
});

test('parseCampaignListResponse rejects missing pagination', () => {
  assert.throws(
    () =>
      parseCampaignListResponse({
        items: [{ id: 'abe62900-7466-5a77-8dac-2cf17fd1dd84', name: 'c1', status: 'draft' }],
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
