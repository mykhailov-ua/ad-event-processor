import { useCallback, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';

import { FilterApplyButton } from '@/shell/action_buttons';
import { CustomerCombobox } from '@/shell/customer_combobox';
import { DirectoryFilterForm, FilterField, FilterPanel } from '@/shell/filter_panel';
import { PageLayout } from '@/shell/page_layout';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from '@/components/ui/checkbox';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { DataFreshness } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import type { CustomerReportConfig } from '@/domains/reports/customer_report_meta';
import {
  buildReportColumnOverviewFields,
  reportRowLabelFromColumn,
} from '@/domains/reports/report_select_overview';
import { DirectorySelectOverviewTable } from '@/shell/directory_select_overview_table';
import {
  directoryOperateRowsIndexed,
  directoryRecordMapIndexed,
} from '@/domains/reports/report_select_overview';
import { adminKit, adminSpacing, adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import { DirectoryStack, MetaLinksBand, TableHost } from '@/shell/ui_bands';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';

export type CustomerReportDirectoryProps<Row> = {
  config: CustomerReportConfig<Row>;
  rows: Row[];
  freshness?: DataFreshness;
  extras?: Record<string, unknown>;
  customerOptions: CustomerComboboxOption[];
  draftCustomerId: string;
  draftFrom: string;
  draftTo: string;
  draftCampaignId: string;
  draftCompare: boolean;
  fetching: boolean;
  listRevalidating?: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  canGoPrev: boolean;
  canGoNext: boolean;
  limit: number;
  offset: number;
  rangeLabel: string;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onDraftCampaignIdChange: (value: string) => void;
  onDraftCompareChange: (value: boolean) => void;
  onApplyFilters: (event?: { preventDefault?: () => void }) => void;
  onPageChange: (nextOffset: number) => void;
};

export function CustomerReportDirectory<Row>({
  config,
  rows,
  freshness,
  extras,
  customerOptions,
  draftCustomerId,
  draftFrom,
  draftTo,
  draftCampaignId,
  draftCompare,
  fetching,
  listRevalidating = false,
  error,
  hasSnapshot,
  canGoPrev,
  canGoNext,
  limit,
  offset,
  rangeLabel,
  onDraftCustomerIdChange,
  onDraftFromChange,
  onDraftToChange,
  onDraftCampaignIdChange,
  onDraftCompareChange,
  onApplyFilters,
  onPageChange,
}: CustomerReportDirectoryProps<Row>) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const nameColumn = config.columns[0];
  const nameColumnLabel = nameColumn?.label ?? 'Dimension';

  const recordById = useMemo(
    () => directoryRecordMapIndexed(rows, (row, index) => config.rowKey(row, index) || null),
    [config, rows]
  );
  const operateRows = useMemo(
    () =>
      directoryOperateRowsIndexed(
        rows,
        (row, index) => config.rowKey(row, index) || null,
        (row) => reportRowLabelFromColumn(row, nameColumn, 'Report row')
      ),
    [config, nameColumn, rows]
  );
  const buildOverviewFields = useCallback(
    (row: Row) =>
      buildReportColumnOverviewFields(row, config.columns, {
        skipColumnIds: nameColumn ? [nameColumn.id] : undefined,
      }),
    [config.columns, nameColumn]
  );

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={3} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title={`Could not load ${config.title}`} message={error.message} />;
  }

  const customerRequired = !draftCustomerId.trim();
  const telemetryMissing =
    typeof extras?.telemetry_missing_rate_display === 'string'
      ? extras.telemetry_missing_rate_display
      : undefined;

  const badge =
    config.badge?.({ freshness }) ??
    (freshness?.stale ? (
      <Badge variant="secondary">Stale data</Badge>
    ) : telemetryMissing ? (
      <span className={adminTypography.bodyMuted}>Telemetry missing: {telemetryMissing}</span>
    ) : freshness?.as_of ? (
      <span className={adminTypography.bodyMuted}>
        As of {displayTimestamp(freshness.as_of)}
      </span>
    ) : null);

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
              <FilterField htmlFor={`${config.key}-customer`} label="Customer" wide>
                <CustomerCombobox
                  id={`${config.key}-customer`}
                  options={customerOptions}
                  value={draftCustomerId}
                  onValueChange={onDraftCustomerIdChange}
                />
              </FilterField>
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
              <FilterField htmlFor={`${config.key}-campaign`} label="Campaign ID">
                <Input
                  id={`${config.key}-campaign`}
                  value={draftCampaignId}
                  onChange={(event) => onDraftCampaignIdChange(event.target.value)}
                />
              </FilterField>
              {config.showCompare ? (
                <FilterField label="Compare">
                  <div className={cn('flex items-center', adminSpacing.gap.md, adminKit.controlHeight)}>
                    <Checkbox
                      checked={draftCompare}
                      id={`${config.key}-compare`}
                      onCheckedChange={(checked) => onDraftCompareChange(checked === true)}
                    />
                    <Label htmlFor={`${config.key}-compare`}>Prior window</Label>
                  </div>
                </FilterField>
              ) : null}
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
            limit={limit}
            pageSizeId={`${config.key}-page-size`}
            rangeLabel={rangeLabel}
            showPrevNext
            onNext={() => onPageChange(offset + limit)}
            onPrev={() => onPageChange(Math.max(0, offset - limit))}
          />
        ) : null
      }
    >
      {customerRequired ? (
        <EmptyState
          title="Customer required"
          description="Pick a customer and date range, then apply filters."
        />
      ) : rows.length === 0 ? (
        <EmptyState title="No rows" description="No rows for the current filters." />
      ) : (
        <DirectoryStack>
          {config.summaryBand?.(extras) ?? null}
          <TableHost className="w-full">
            <DirectorySelectOverviewTable
              buildOverviewFields={buildOverviewFields}
              disabled={fetching}
              nameColumnLabel={nameColumnLabel}
              overviewTitle={(row) => String(reportRowLabelFromColumn(row, nameColumn, config.title))}
              recordById={recordById}
              revalidating={listRevalidating}
              rows={operateRows}
              selectedId={selectedId}
              onSelectedIdChange={setSelectedId}
            />
          </TableHost>
        </DirectoryStack>
      )}
    </PageLayout>
  );
}
