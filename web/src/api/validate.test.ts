import assert from 'node:assert/strict';
import test from 'node:test';

import { ApiError } from '@/api/api_error';
import {
  parseAuditLogRow,
  parseCampaignBulkActionResponse,
  parseCampaignFlowValidateResponse,
  parseCampaignListResponse,
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
