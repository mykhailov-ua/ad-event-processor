import assert from 'node:assert/strict';
import test from 'node:test';

import { ApiError } from '@/api/api_error';
import { exportHubErrorMessage } from '@/domains/exports/export_hub_errors';
import { invalidateReportCatalogCache } from '@/lib/report_catalog_cache';

test('exportHubErrorMessage maps catalog server failure', () => {
  const message = exportHubErrorMessage(
    new ApiError(503, 'INTERNAL_ERROR', 'report catalog unavailable')
  );
  assert.match(message, /server could not complete/i);
});

test('invalidateReportCatalogCache is safe to call before catalog retry', () => {
  assert.doesNotThrow(() => {
    invalidateReportCatalogCache();
  });
});
