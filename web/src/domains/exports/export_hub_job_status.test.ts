import assert from 'node:assert/strict';
import test from 'node:test';

import {
  exportJobCanDownload,
  exportJobPhase,
  exportJobStatusDisplayLabel,
  formatExportJobBytes,
  formatExportJobRecentSummary,
  formatExportJobRowSummary,
  truncateExportJobInlineText,
} from '@/domains/exports/export_hub_job_status';

test('exportJobStatusDisplayLabel avoids unknown placeholder for empty status', () => {
  assert.equal(exportJobStatusDisplayLabel(undefined), 'No status yet');
  assert.equal(exportJobStatusDisplayLabel(''), 'No status yet');
  assert.equal(exportJobStatusDisplayLabel('completed'), 'Completed');
});

test('exportJobPhase maps running states to pending', () => {
  assert.equal(exportJobPhase('RUNNING'), 'pending');
  assert.equal(exportJobPhase('processing'), 'pending');
});

test('exportJobCanDownload is true only for completed jobs', () => {
  assert.equal(exportJobCanDownload('COMPLETED'), true);
  assert.equal(exportJobCanDownload('running'), false);
});

test('formatExportJobRowSummary combines bytes and row limit', () => {
  const summary = formatExportJobRowSummary(1_200_000, 44_892_928);
  assert.match(summary ?? '', /MB/);
  assert.match(summary ?? '', /1,200,000 row limit/);
});

test('formatExportJobBytes uses binary units', () => {
  assert.equal(formatExportJobBytes(1024), '1.0 KB');
});

test('formatExportJobRecentSummary hides row limit on failed jobs', () => {
  const summary = formatExportJobRecentSummary('failed', 1_200_000, 44_892_928);
  assert.match(summary ?? '', /MB/);
  assert.doesNotMatch(summary ?? '', /row limit/i);
});

test('formatExportJobRecentSummary keeps row limit for completed jobs', () => {
  const summary = formatExportJobRecentSummary('completed', 1_200_000, 44_892_928);
  assert.match(summary ?? '', /1,200,000 row limit/);
});

test('truncateExportJobInlineText adds ellipsis and full text for tooltip', () => {
  const long = 'a'.repeat(140);
  const inline = truncateExportJobInlineText(long, 40);
  assert.equal(inline.truncated, true);
  assert.equal(inline.full.length, 140);
  assert.match(inline.display, /\.\.\.$/);
  assert.equal(inline.display.length, 40);
});
