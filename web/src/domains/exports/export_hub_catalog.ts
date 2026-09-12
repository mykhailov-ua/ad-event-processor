import type { ReportCatalogRow } from '@/api/types';
import { buildExportHubHref } from '@/lib/export_hub_paths';
import { resolveReportDisplayTitle } from '@/lib/report_paths';

export type ExportHubKind = 'report' | 'billing' | 'audit' | 'directory';

export type ExportHubEntry = {
  id: string;
  title: string;
  description: string;
  kind: ExportHubKind;
  defaultFormat: string;
  reportKey?: string;
  /** In-app route when kind is directory (client-side export on that page). */
  href?: string;
};

const REPORT_EXPORT_KEY_FALLBACK = [
  'placements',
  'campaign-stats',
  'fraud-evidence-pack-bulk',
  'customer-portfolio',
  'true-roi',
] as const;

export const EXPORT_HUB_CUSTOM_PREFIX = '';

const STATIC_EXPORT_HUB_ENTRIES: ExportHubEntry[] = [
  {
    id: 'billing-ledger',
    title: 'Billing ledger',
    description:
      'Async billing ledger export for a customer and date range. Customer detail Ledger tab Export CSV is a one-shot balance snapshot only; use this job for full ledger history.',
    kind: 'billing',
    defaultFormat: 'csv',
  },
  {
    id: 'audit-csv',
    title: 'Audit log CSV',
    description: 'One-shot CSV export of audit events (optional PII redaction).',
    kind: 'audit',
    defaultFormat: 'csv',
  },
  {
    id: 'campaigns-directory',
    title: 'Campaigns list CSV',
    description:
      'Export visible campaigns from the directory toolbar (client-side CSV of loaded rows).',
    kind: 'directory',
    defaultFormat: 'csv',
    href: '/campaigns',
  },
  {
    id: 'customers-directory',
    title: 'Customer portfolio export',
    description:
      'Async customer portfolio KPI export for the selected customer and date range. Customer directory is operational CRUD; use Export hub for portfolio jobs.',
    kind: 'report',
    defaultFormat: 'csv',
    reportKey: 'customer-portfolio',
  },
];

function reportEntryFromKey(key: string, row?: ReportCatalogRow): ExportHubEntry {
  return {
    id: `report-${key}`,
    title: resolveReportDisplayTitle(key, row?.title),
    description:
      row?.description?.trim() ||
      'Enqueue an async report export job for the selected customer and date range.',
    kind: 'report',
    defaultFormat: 'csv',
    reportKey: key,
  };
}

export function exportHubEntriesFromCatalog(rows?: ReportCatalogRow[]): ExportHubEntry[] {
  const catalogRows = rows ?? [];
  const reportEntries =
    catalogRows.length > 0
      ? catalogRows
          .filter((row) => row.key?.trim())
          .map((row) => reportEntryFromKey(row.key!.trim(), row))
      : REPORT_EXPORT_KEY_FALLBACK.map((key) => reportEntryFromKey(key));
  return [...reportEntries, ...STATIC_EXPORT_HUB_ENTRIES];
}

export function exportHubCatalogOptionValue(entry: ExportHubEntry): string {
  return entry.id;
}

export function exportHubCatalogPickerLabel(entry: ExportHubEntry): string {
  return `${EXPORT_HUB_KIND_LABELS[entry.kind]}: ${entry.title}`;
}

export function exportHubCatalogItemLabel(entry: ExportHubEntry): string {
  return entry.title;
}

export type ExportHubCatalogOption = {
  value: string;
  label: string;
};

export function exportHubCatalogOptions(entries: ExportHubEntry[]): ExportHubCatalogOption[] {
  return entries
    .map((entry) => ({
      value: exportHubCatalogOptionValue(entry),
      label: exportHubCatalogItemLabel(entry),
    }))
    .sort((left, right) => left.label.localeCompare(right.label, undefined, { sensitivity: 'base' }));
}

export function resolveExportHubCatalogValue(
  value: string,
  entries: ExportHubEntry[]
): { entry?: ExportHubEntry; customReportKey?: string } {
  const trimmed = value.trim();
  if (!trimmed) {
    return {};
  }
  if (trimmed.startsWith(EXPORT_HUB_CUSTOM_PREFIX)) {
    const customReportKey = trimmed.slice(EXPORT_HUB_CUSTOM_PREFIX.length).trim();
    return customReportKey ? { customReportKey } : {};
  }
  const byId = entries.find((entry) => entry.id === trimmed);
  if (byId) {
    return { entry: byId };
  }
  const byReportKey = entries.find((entry) => entry.reportKey === trimmed);
  if (byReportKey) {
    return { entry: byReportKey };
  }
  return { customReportKey: trimmed };
}

export const EXPORT_HUB_ENTRIES: ExportHubEntry[] = exportHubEntriesFromCatalog();

export function exportHubEntryHref(entry: ExportHubEntry): string {
  if (entry.kind === 'directory' && entry.href?.trim()) {
    return entry.href.trim();
  }
  return buildExportHubHref({
    entry: entry.id,
    kind: entry.kind,
    reportKey: entry.reportKey,
    format: entry.defaultFormat,
  });
}

export function findExportHubEntry(
  entryId: string,
  catalogRows?: ReportCatalogRow[]
): ExportHubEntry | undefined {
  const trimmed = entryId.trim();
  if (!trimmed) {
    return undefined;
  }
  return exportHubEntriesFromCatalog(catalogRows).find((entry) => entry.id === trimmed);
}

export const EXPORT_HUB_KIND_LABELS: Record<ExportHubKind, string> = {
  report: 'Reports',
  billing: 'Billing',
  audit: 'Audit',
  directory: 'Directory',
};
