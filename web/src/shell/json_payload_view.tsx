import { memo, useMemo } from 'react';

import { PanelSection } from '@/shell/stat_panel';
import { Badge } from '@/components/ui/badge';
import { ReportMapTable } from '@/shell/report_map_table';
import { deriveColumns, formatMapCell } from '@/lib/report_table';
import type { JsonPayload } from '@/lib/json_record';
import { isJsonRecord, toJsonRecord } from '@/lib/json_record';
import { cn } from '@/lib/utils';

const DASHBOARD_TABLE_ROW_CAP = 100;

type DashboardTableMeta = {
  truncated?: boolean;
  total?: number;
};

type DashboardTableSection = {
  key: string;
  rows: Record<string, unknown>[];
  columns: string[];
  truncated: boolean;
  totalCount: number;
};

function readTableSectionMeta(
  payload: Record<string, unknown>,
  key: string
): DashboardTableMeta | undefined {
  const sectionsMeta = payload.table_sections_meta;
  if (!isJsonRecord(sectionsMeta)) {
    return undefined;
  }
  const meta = sectionsMeta[key];
  if (!isJsonRecord(meta)) {
    return undefined;
  }
  return {
    truncated: meta.truncated === true,
    total: typeof meta.total === 'number' ? meta.total : undefined,
  };
}

function partitionPayload(payload: JsonPayload): {
  scalarEntries: Array<[string, unknown]>;
  tableSections: DashboardTableSection[];
} {
  const record = toJsonRecord(payload);
  const scalarEntries: Array<[string, unknown]> = [];
  const tableSections: DashboardTableSection[] = [];

  for (const [key, value] of Object.entries(record)) {
    if (key === 'table_sections_meta') {
      continue;
    }
    if (Array.isArray(value) && value.length > 0 && isJsonRecord(value[0])) {
      const rows = value as Record<string, unknown>[];
      const serverMeta = readTableSectionMeta(record, key);
      const totalCount = serverMeta?.total ?? rows.length;
      const serverTruncated = serverMeta?.truncated === true;
      const clientTruncated = !serverTruncated && rows.length > DASHBOARD_TABLE_ROW_CAP;
      const visibleRows = clientTruncated ? rows.slice(0, DASHBOARD_TABLE_ROW_CAP) : rows;
      tableSections.push({
        key,
        rows: visibleRows,
        columns: deriveColumns(visibleRows),
        truncated: serverTruncated || clientTruncated,
        totalCount: clientTruncated ? rows.length : totalCount,
      });
      continue;
    }
    if (isJsonRecord(value)) {
      scalarEntries.push([key, value]);
      continue;
    }
    if (typeof value !== 'object') {
      scalarEntries.push([key, value]);
    }
  }

  return { scalarEntries, tableSections };
}

export type JsonPayloadViewProps = {
  payload: JsonPayload;
  formatKey?: (key: string) => string;
  formatColumn?: (key: string) => string;
};

export const JsonPayloadView = memo(function JsonPayloadView({
  payload,
  formatKey,
  formatColumn,
}: JsonPayloadViewProps) {
  const { scalarEntries, tableSections } = useMemo(() => partitionPayload(payload), [payload]);

  return (
    <div className="grid gap-4">
      {scalarEntries.length > 0 ? (
        <div className="grid gap-4 sm:grid-cols-[repeat(auto-fit,minmax(220px,1fr))]">
          {scalarEntries.map(([key, value]) => (
            <div className="ui-surface-raised grid gap-2 p-5" key={key}>
              <p className="text-sm font-medium text-muted-foreground">
                {formatKey ? formatKey(key) : key}
              </p>
              <div
                className={cn(
                  'text-sm',
                  (typeof value === 'object' && value != null) || key.includes('template')
                    ? 'break-all font-mono text-xs'
                    : 'tabular-nums'
                )}
              >
                {formatMapCell(value)}
              </div>
            </div>
          ))}
        </div>
      ) : null}

      {tableSections.map((section) => (
        <PanelSection
          key={section.key}
          meta={
            section.truncated ? (
              <Badge variant="secondary">
                showing first {section.rows.length} of {section.totalCount}
              </Badge>
            ) : null
          }
          title={formatKey ? formatKey(section.key) : section.key}
        >
          <ReportMapTable
            className="border-0 bg-transparent shadow-none"
            columns={section.columns}
            formatColumn={formatColumn ?? formatKey}
            rowKeyPrefix={section.key}
            rows={section.rows}
          />
        </PanelSection>
      ))}
    </div>
  );
});

/** @deprecated Import JsonPayloadView from @/shell/json_payload_view */
export const JsonDashboardView = JsonPayloadView;
