import { Link } from 'react-router-dom';

import { FilterApplyButton } from '@/shell/action_buttons';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
  directoryTableRevalidatingClass,
} from '@/shell/directory_table';
import { DirectoryFilterForm, FilterField, FilterPanel } from '@/shell/filter_panel';
import { PageLayout } from '@/shell/page_layout';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import { Badge } from '@/components/ui/badge';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import type { DataFreshness } from '@/api/types';
import type { MlReportConfig } from '@/domains/reports/ml_report_meta';
import { DirectoryStack, MetaLinksBand, TableHost } from '@/shell/ui_bands';
import { cn } from '@/lib/utils';

export type MlReportDirectoryProps<Row> = {
  config: MlReportConfig<Row>;
  rows: Row[];
  freshness?: DataFreshness;
  draftFrom: string;
  draftTo: string;
  fetching: boolean;
  listRevalidating?: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  rangeReady: boolean;
  canGoPrev: boolean;
  canGoNext: boolean;
  limit: number;
  offset: number;
  rangeLabel: string;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onApplyFilters: (event?: { preventDefault?: () => void }) => void;
  onPageChange: (nextOffset: number) => void;
};

export function MlReportDirectory<Row>({
  config,
  rows,
  freshness,
  draftFrom,
  draftTo,
  fetching,
  listRevalidating = false,
  error,
  hasSnapshot,
  rangeReady,
  canGoPrev,
  canGoNext,
  limit,
  offset,
  rangeLabel,
  onDraftFromChange,
  onDraftToChange,
  onApplyFilters,
  onPageChange,
}: MlReportDirectoryProps<Row>) {
  const columnCount = config.columns.length;

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={Math.min(columnCount, 8)} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title={`Could not load ${config.title}`} message={error.message} />;
  }

  const badge = freshness?.stale ? (
    <Badge variant="secondary">Stale data</Badge>
  ) : freshness?.ch_lag_seconds != null ? (
    <span className="text-xs text-muted-foreground">CH lag {freshness.ch_lag_seconds}s</span>
  ) : null;

  return (
    <PageLayout
      badge={badge}
      controlPanel={
        <DirectoryStack>
          <MetaLinksBand>
            <Link to="/reports">Reports catalog</Link>
          </MetaLinksBand>
          <FilterPanel>
            <DirectoryFilterForm layout="auto-fill" onSubmit={onApplyFilters}>
              <FilterField htmlFor={`${config.key}-from`} label="From">
                <DatetimePicker
                  id={`${config.key}-from`}
                  label="From"
                  value={draftFrom}
                  onChange={onDraftFromChange}
                />
              </FilterField>
              <FilterField htmlFor={`${config.key}-to`} label="To">
                <DatetimePicker
                  id={`${config.key}-to`}
                  label="To"
                  value={draftTo}
                  onChange={onDraftToChange}
                />
              </FilterField>
              <FilterApplyButton disabled={fetching} type="submit" />
            </DirectoryFilterForm>
          </FilterPanel>
        </DirectoryStack>
      }
      description={config.description}
      title={config.title}
      footer={
        rows.length > 0 ? (
          <DirectoryPaginationFooter
            canGoNext={canGoNext}
            canGoPrev={canGoPrev}
            disabled={fetching}
            rangeLabel={rangeLabel}
            onNext={() => onPageChange(offset + limit)}
            onPrev={() => onPageChange(Math.max(0, offset - limit))}
          />
        ) : null
      }
    >
      {!rangeReady ? (
        <EmptyState title="Date range required" description="Set from and to, then apply." />
      ) : rows.length === 0 && !fetching ? (
        <EmptyState title="No rows" description="No ML report data for this window." />
      ) : (
        <TableHost className="w-full">
          <DirectoryTable>
            <DirectoryTableHead>
              <TableRow>
                {config.columns.map((column) => (
                  <TableHeader
                    key={column.id}
                    className={column.align === 'end' ? 'text-right' : undefined}
                  >
                    {column.label}
                  </TableHeader>
                ))}
              </TableRow>
            </DirectoryTableHead>
            <TableBody className={cn(listRevalidating && directoryTableRevalidatingClass)}>
              {rows.map((row, index) => (
                <TableRow key={config.rowKey(row, index)}>
                  {config.columns.map((column) => (
                    <TableCell
                      key={column.id}
                      className={column.align === 'end' ? 'text-right ' : undefined}
                    >
                      {column.cell(row)}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </DirectoryTable>
        </TableHost>
      )}
    </PageLayout>
  );
}
