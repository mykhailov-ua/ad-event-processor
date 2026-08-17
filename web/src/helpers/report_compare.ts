import type { ReportCompareDeltas, ReportRow } from '../types/api/report.js';
import { formatAmountMicro } from './money.js';
import { t } from './i18n.js';

export function reportCompareLabel(): string {
  return t('report.compare', 'Compare with previous period');
}

export function rowCompareDeltas(row: ReportRow): ReportCompareDeltas | null {
  const compare = row.compare;
  if (!compare || typeof compare !== 'object') return null;
  return compare as ReportCompareDeltas;
}

export function formatSpendDelta(row: ReportRow): string {
  const delta = rowCompareDeltas(row)?.spend_micro_delta;
  if (delta == null) return '—';
  return formatAmountMicro(delta);
}
