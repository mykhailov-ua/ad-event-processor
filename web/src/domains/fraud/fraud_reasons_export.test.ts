import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildFraudReasonsExportCsv,
  exportColumnsForReportKey,
  fraudReasonExportCellValue,
} from './fraud_reasons_export.ts';

test('exportColumnsForReportKey includes placement only on fraud-breakdown', () => {
  const detailed = exportColumnsForReportKey('fraud-breakdown');
  const wire = exportColumnsForReportKey('wire-signal-breakdown');
  assert.ok(detailed.includes('placement_id'));
  assert.ok(!wire.includes('placement_id'));
  assert.ok(wire.includes('signals_degraded'));
});

test('buildFraudReasonsExportCsv escapes reason commas', () => {
  const csv = buildFraudReasonsExportCsv(
    ['campaign_id', 'fraud_reason', 'event_count'],
    [
      {
        campaign_id: '00000000-0000-4000-8000-000000000001',
        fraud_reason: 'tls_ja4_mismatch, h2',
        event_count: 42,
      },
    ]
  );
  assert.match(csv, /"tls_ja4_mismatch, h2"/);
  assert.equal(fraudReasonExportCellValue('silent_reject_ratio', { silent_reject_ratio: 0.125 }), '12.5%');
});
