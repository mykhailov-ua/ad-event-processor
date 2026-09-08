import type { FraudReasonsReportKey } from '@/api/types';

export type { FraudReasonRow } from '@/api/types';

export { type FraudReasonsReportKey };

export const FRAUD_REASONS_REPORT_META: Record<
  FraudReasonsReportKey,
  { title: string; description: string; exportFilename: string }
> = {
  'fraud-breakdown': {
    title: 'Fraud reasons',
    description: 'Fraud events by reason and placement',
    exportFilename: 'fraud-reasons.csv',
  },
  'wire-signal-breakdown': {
    title: 'Wire fraud signals',
    description: 'L7, TLS, and HTTP/2 wire fraud signals',
    exportFilename: 'wire-fraud-signals.csv',
  },
};
