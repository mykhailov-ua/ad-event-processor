import type { ReactNode } from 'react';

import type { FraudCatalogReportRow } from '@/api/types';
import type {
  FraudCatalogColumnDef,
  FraudCatalogColumnKind,
} from '@/domains/fraud/fraud_catalog_report_meta';
import { displayCount } from '@/lib/display';
import { Badge } from '@/components/ui/badge';

function readCellValue(row: FraudCatalogReportRow, column: FraudCatalogColumnDef): unknown {
  const displayField = column.displayField ?? column.field;
  const displayValue = row[displayField];
  if (displayValue != null && String(displayValue).trim() !== '') {
    return displayValue;
  }
  return row[column.field];
}

function formatRatio(value: unknown): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return '';
  }
  return `${(value * 100).toFixed(1)}%`;
}

function formatPercent(value: unknown): string {
  if (typeof value === 'string' && value.trim() !== '') {
    return value;
  }
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    return '';
  }
  return `${value.toFixed(1)}%`;
}

export function formatFraudCatalogCell(
  row: FraudCatalogReportRow,
  column: FraudCatalogColumnDef
): string {
  const value = readCellValue(row, column);
  switch (column.kind) {
    case 'count':
      return typeof value === 'number' ? displayCount(value) : '';
    case 'ratio':
      return formatRatio(value);
    case 'percent':
      return formatPercent(value);
    case 'badge':
    case 'mono':
    case 'text':
    default:
      return value == null ? '' : String(value);
  }
}

export function renderFraudCatalogCell(
  row: FraudCatalogReportRow,
  column: FraudCatalogColumnDef
): ReactNode {
  const text = formatFraudCatalogCell(row, column);
  if (column.kind === 'badge' && text.trim() !== '') {
    const tone = 'delta_tone' in row ? row.delta_tone : undefined;
    const variant =
      tone === 'up' || tone === 'warn'
        ? 'secondary'
        : tone === 'down' || tone === 'good'
          ? 'default'
          : 'outline';
    return <Badge variant={variant}>{text}</Badge>;
  }
  if (column.kind === 'mono') {
    return <span className="text-xs">{text || '-'}</span>;
  }
  return text || '-';
}

export function fraudCatalogColumnClass(kind: FraudCatalogColumnKind): string | undefined {
  if (kind === 'mono') {
    return 'text-xs';
  }
  return undefined;
}
