import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';

import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { PanelSection } from '@/shell/stat_panel';
import type { DashboardBreakdownRow, DashboardBreakdownTable } from '@/domains/dashboards/buyer_dashboard_types';
import {
  formatDashboardCrPct,
  formatDashboardUsdFromMicro,
} from '@/domains/dashboards/dashboard_format';
import { formatRoi } from '@/domains/dashboards/dashboard_metrics';
import type { DashboardBreakdownColumnId } from '@/domains/dashboards/dashboard_preferences';
import { BREAKDOWN_COLUMN_LABELS } from '@/domains/dashboards/dashboard_preferences';
import { displayCount } from '@/lib/display';

export type DashboardBreakdownTableProps = {
  title: string;
  table: DashboardBreakdownTable | undefined;
  columns: DashboardBreakdownColumnId[];
  nameLink?: (row: { id?: string; name?: string }) => ReactNode;
  emptyLabel?: string;
};

type BreakdownCellContext = {
  row: DashboardBreakdownRow;
  totals?: DashboardBreakdownRow;
  isTotal?: boolean;
};

function renderBreakdownCell(columnId: DashboardBreakdownColumnId, ctx: BreakdownCellContext): ReactNode {
  const { row } = ctx;
  switch (columnId) {
    case 'name':
      return ctx.isTotal ? 'Total' : row.name;
    case 'clicks':
      return displayCount(row.clicks);
    case 'unique_clicks':
      return displayCount(row.unique_clicks);
    case 'conversions':
      return displayCount(row.conversions);
    case 'cost':
      return formatDashboardUsdFromMicro(row.cost_micro);
    case 'revenue':
      return formatDashboardUsdFromMicro(row.revenue_micro);
    case 'profit':
      return formatDashboardUsdFromMicro(row.profit_micro);
    case 'cpc':
      return formatDashboardUsdFromMicro(row.cpc_micro);
    case 'cpa':
      return formatDashboardUsdFromMicro(row.cpa_micro);
    case 'cr':
      return formatDashboardCrPct(row.cr_pct);
    case 'epc':
      return formatDashboardUsdFromMicro(row.epc_micro);
    case 'roi':
      return formatRoi(row.roi_pct);
    default:
      return '';
  }
}

export function DashboardBreakdownTableSection({
  title,
  table,
  columns,
  nameLink,
  emptyLabel = 'No data in this range.',
}: DashboardBreakdownTableProps) {
  const rows = table?.rows ?? [];
  const totals = table?.totals;
  const visibleColumns = columns.length > 0 ? columns : (['name', 'clicks', 'conversions'] as DashboardBreakdownColumnId[]);

  return (
    <PanelSection className="min-w-0" title={title}>
      {rows.length === 0 ? (
        <p className="px-5 py-6 text-sm text-muted-foreground">{emptyLabel}</p>
      ) : (
        <DirectoryTable className="border-0 bg-transparent shadow-none" scrollable>
          <TableHeader>
            <TableRow>
              {visibleColumns.map((columnId) => (
                <DirectoryTableHead
                  key={columnId}
                  align={columnId === 'name' ? 'start' : 'end'}
                  className={columnId === 'name' ? 'min-w-[8rem]' : 'min-w-[4.5rem]'}
                >
                  {BREAKDOWN_COLUMN_LABELS[columnId]}
                </DirectoryTableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row) => (
              <TableRow key={row.id ?? row.name}>
                {visibleColumns.map((columnId) => (
                  <TableCell
                    key={columnId}
                    className={
                      columnId === 'name'
                        ? 'max-w-[10rem] truncate font-medium'
                        : 'font-numeric text-right tabular-nums'
                    }
                  >
                    {columnId === 'name' ? (
                      nameLink ? (
                        nameLink(row)
                      ) : (
                        <span className="block truncate" title={row.name}>
                          {row.name}
                        </span>
                      )
                    ) : (
                      renderBreakdownCell(columnId, { row })
                    )}
                  </TableCell>
                ))}
              </TableRow>
            ))}
            {totals ? (
              <TableRow className="font-medium">
                {visibleColumns.map((columnId) => (
                  <TableCell
                    key={columnId}
                    className={
                      columnId === 'name'
                        ? 'max-w-[10rem] truncate'
                        : 'font-numeric text-right tabular-nums'
                    }
                  >
                    {columnId === 'name' ? (
                      'Total'
                    ) : (
                      renderBreakdownCell(columnId, { row: totals, isTotal: true })
                    )}
                  </TableCell>
                ))}
              </TableRow>
            ) : null}
          </TableBody>
        </DirectoryTable>
      )}
      {table?.truncated ? (
        <p className="px-5 pb-4 text-xs text-muted-foreground">
          Showing top {rows.length} of {table.total ?? rows.length} rows.
        </p>
      ) : null}
    </PanelSection>
  );
}

export function campaignBreakdownLink(row: { id?: string; name?: string }) {
  if (!row.id) {
    return (
      <span className="block truncate" title={row.name}>
        {row.name}
      </span>
    );
  }
  return (
    <Link
      className="block truncate text-primary hover:underline"
      title={row.name}
      to={`/campaigns/${row.id}/edit`}
    >
      {row.name}
    </Link>
  );
}
