import { getFraudReasonsReport } from '@/api/reports_api';
import type { ReportRunQuery, FraudReasonRow, FraudReasonsReportKey } from '@/api/types';

export const FRAUD_REASONS_EXPORT_PAGE_SIZE = 1000;
export const FRAUD_REASONS_EXPORT_MAX_ROWS = 5000;

export type FraudReasonsExportColumnId =
  | 'campaign_id'
  | 'fraud_reason'
  | 'fraud_category_label'
  | 'placement_id'
  | 'event_count'
  | 'silent_reject_count'
  | 'silent_reject_ratio'
  | 'signals_degraded';

export const FRAUD_REASONS_EXPORT_COLUMNS: ReadonlyArray<{
  id: FraudReasonsExportColumnId;
  label: string;
}> = [
  { id: 'campaign_id', label: 'Campaign ID' },
  { id: 'fraud_reason', label: 'Fraud reason' },
  { id: 'fraud_category_label', label: 'Category' },
  { id: 'placement_id', label: 'Placement ID' },
  { id: 'event_count', label: 'Events' },
  { id: 'silent_reject_count', label: 'Non-blocking' },
  { id: 'silent_reject_ratio', label: 'Non-blocking ratio' },
  { id: 'signals_degraded', label: 'Signals degraded' },
];

function csvEscape(value: string): string {
  return `"${value.replaceAll('"', '""')}"`;
}

function formatRatio(value?: number): string {
  if (value == null || !Number.isFinite(value)) {
    return '';
  }
  return `${(value * 100).toFixed(1)}%`;
}

export function fraudReasonExportCellValue(
  columnId: FraudReasonsExportColumnId,
  row: FraudReasonRow
): string {
  switch (columnId) {
    case 'campaign_id':
      return row.campaign_id ?? '';
    case 'fraud_reason':
      return row.fraud_reason ?? '';
    case 'fraud_category_label':
      return row.fraud_category_label ?? row.fraud_category ?? '';
    case 'placement_id':
      return row.placement_id ?? '';
    case 'event_count':
      return row.event_count != null ? String(row.event_count) : '';
    case 'silent_reject_count':
      return row.silent_reject_count != null ? String(row.silent_reject_count) : '';
    case 'silent_reject_ratio':
      return formatRatio(row.silent_reject_ratio);
    case 'signals_degraded':
      return row.signals_degraded ? 'yes' : '';
    default:
      return '';
  }
}

export function exportColumnsForReportKey(
  reportKey: FraudReasonsReportKey
): ReadonlyArray<FraudReasonsExportColumnId> {
  const base: FraudReasonsExportColumnId[] = [
    'campaign_id',
    'fraud_reason',
    'fraud_category_label',
    'event_count',
    'silent_reject_count',
    'silent_reject_ratio',
  ];
  if (reportKey === 'fraud-breakdown') {
    return [...base.slice(0, 3), 'placement_id', ...base.slice(3)];
  }
  return [...base, 'signals_degraded'];
}

export function buildFraudReasonsExportCsv(
  columns: ReadonlyArray<FraudReasonsExportColumnId>,
  rows: ReadonlyArray<FraudReasonRow>
): string {
  const labels = columns.map(
    (columnId) =>
      FRAUD_REASONS_EXPORT_COLUMNS.find((column) => column.id === columnId)?.label ?? columnId
  );
  const lines = rows.map((row) =>
    columns.map((columnId) => csvEscape(fraudReasonExportCellValue(columnId, row))).join(',')
  );
  return [labels.join(','), ...lines].join('\n');
}

function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

export type FraudReasonsExportDataset = {
  rows: FraudReasonRow[];
  truncated: boolean;
};

export async function listAllFraudReasonRowsForExport(
  reportKey: FraudReasonsReportKey,
  params: ReportRunQuery,
  signal?: AbortSignal
): Promise<FraudReasonsExportDataset> {
  const rows: FraudReasonRow[] = [];
  let offset = params.offset ?? 0;

  while (rows.length < FRAUD_REASONS_EXPORT_MAX_ROWS) {
    const page = await getFraudReasonsReport(
      reportKey,
      {
        ...params,
        limit: FRAUD_REASONS_EXPORT_PAGE_SIZE,
        offset,
      },
      signal
    );
    if (page.rows.length === 0) {
      break;
    }
    rows.push(...page.rows);
    if (!page.next_cursor) {
      break;
    }
    offset += page.rows.length;
    if (rows.length >= FRAUD_REASONS_EXPORT_MAX_ROWS) {
      break;
    }
  }

  return {
    rows: rows.slice(0, FRAUD_REASONS_EXPORT_MAX_ROWS),
    truncated: rows.length >= FRAUD_REASONS_EXPORT_MAX_ROWS,
  };
}

export function exportFraudReasonRowsCsv(
  reportKey: FraudReasonsReportKey,
  rows: ReadonlyArray<FraudReasonRow>
): void {
  if (rows.length === 0) {
    return;
  }
  const columns = exportColumnsForReportKey(reportKey);
  const csv = buildFraudReasonsExportCsv(columns, rows);
  const filename = reportKey === 'wire-signal-breakdown'
    ? 'wire-fraud-signals.csv'
    : 'fraud-reasons.csv';
  downloadBlob(new Blob([csv], { type: 'text/csv;charset=utf-8' }), filename);
}
