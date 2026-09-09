import { useCallback, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';

import type { ReportCompareDeltas, SourceQualityGroupBy, SourceQualityRow } from '@/api/types';
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
import type { DirectoryOverviewField } from '@/shell/directory_overview_dialog';
import {
  DirectorySelectOverviewTable,
  directoryOperateRows,
  directoryRecordMap,
} from '@/shell/directory_select_overview_table';
import { displayCount, displayMicro } from '@/lib/display';
import { resolveEconomicsProfitMicro, resolveEconomicsRoiPct } from '@/lib/economics';
import { adminKit, adminSpacing, adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import { DirectoryStack, MetaLinksBand, TableHost } from '@/shell/ui_bands';
import { useSourceQualityPageWorkspace } from '@/domains/reports/use_source_quality_page_workspace';

const GROUP_BY_OPTIONS: { id: SourceQualityGroupBy; label: string }[] = [
  { id: 'placement', label: 'Placement' },
  { id: 'campaign', label: 'Campaign' },
  { id: 'country', label: 'Country' },
  { id: 'city', label: 'City' },
  { id: 'device', label: 'Device' },
  { id: 'sub_id', label: 'Sub ID' },
];

function formatPct(value?: number): string {
  if (value == null || !Number.isFinite(value)) {
    return '-';
  }
  return `${value.toFixed(1)}%`;
}

function formatRatio(value?: number): string {
  if (value == null || !Number.isFinite(value)) {
    return '-';
  }
  return `${(value * 100).toFixed(2)}%`;
}

function deltaSuffix(delta?: number): string {
  if (delta == null || delta === 0) {
    return '';
  }
  const sign = delta > 0 ? '+' : '';
  return ` (${sign}${displayCount(delta)})`;
}

function metricOverviewValue(
  value: number | undefined,
  delta?: number,
  format: 'count' | 'micro' = 'count'
): string {
  const text = format === 'micro' ? displayMicro(value) || '-' : displayCount(value) || '-';
  const suffix = deltaSuffix(delta);
  return suffix ? `${text}${suffix}` : text;
}

function sourceQualityRowId(row: SourceQualityRow): string {
  const parts = [
    row.placement_id,
    row.campaign_id,
    row.country,
    row.city,
    row.device,
    row.sub1,
  ].filter(Boolean);
  return parts.length > 0 ? parts.join('|') : '';
}

function sourceQualityRowLabel(row: SourceQualityRow): string {
  return (
    row.placement_id ??
    row.campaign_id ??
    row.country ??
    row.city ??
    row.device ??
    row.sub1 ??
    'Source row'
  );
}

function sourceQualityNameColumnLabel(draftGroupBy: SourceQualityGroupBy[]): string {
  const primary = GROUP_BY_OPTIONS.find((option) => draftGroupBy.includes(option.id));
  return primary?.label ?? 'Source';
}

function buildSourceQualityOverviewFields(
  row: SourceQualityRow,
  detailMode: boolean
): DirectoryOverviewField[] {
  const compare: ReportCompareDeltas | undefined = row.compare;
  const fields: DirectoryOverviewField[] = [
    {
      label: 'Placement',
      value: (
        <span className={cn(adminTypography.monoData, 'text-muted-foreground')}>
          {row.placement_id ?? '-'}
        </span>
      ),
    },
    {
      label: 'Campaign',
      value: (
        <span className={cn(adminTypography.monoData, 'text-muted-foreground')}>
          {row.campaign_id ?? '-'}
        </span>
      ),
    },
  ];
  if (detailMode) {
    fields.push(
      { label: 'Country', value: row.country ?? '-' },
      { label: 'City', value: row.city ?? '-' },
      { label: 'Device', value: row.device ?? '-' },
      {
        label: 'Sub',
        value: (
          <span className={cn(adminTypography.monoData, 'text-muted-foreground')}>
            {row.sub1 ?? '-'}
          </span>
        ),
      }
    );
  }
  fields.push(
    {
      label: 'Impressions',
      value: metricOverviewValue(row.impressions, compare?.impressions_delta),
    },
    { label: 'Clicks', value: metricOverviewValue(row.clicks, compare?.clicks_delta) },
    {
      label: 'Conversions',
      value: metricOverviewValue(row.conversions, compare?.conversions_delta),
    },
    {
      label: 'Spend',
      value: metricOverviewValue(row.spend_micro, compare?.spend_micro_delta, 'micro'),
    },
    {
      label: 'Revenue',
      value: metricOverviewValue(row.revenue_micro, compare?.revenue_micro_delta, 'micro'),
    },
    { label: 'Profit', value: displayMicro(resolveEconomicsProfitMicro(row)) || '-' },
    { label: 'ROI', value: formatPct(resolveEconomicsRoiPct(row)) },
    { label: 'CPA', value: displayMicro(row.cpa_micro) || '-' },
    { label: 'CTR', value: formatRatio(row.ctr) },
    { label: 'IVT', value: formatRatio(row.ivt_rate) }
  );
  return fields;
}

export function SourceQualityDirectory() {
  const {
    rows,
    freshness,
    customerOptions,
    draftCustomerId,
    draftFrom,
    draftTo,
    draftCampaignId,
    draftCompare,
    draftGroupBy,
    detailMode,
    fetching,
    listRevalidating,
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
    onToggleGroupBy,
    onApplyFilters,
    onPageChange,
  } = useSourceQualityPageWorkspace();

  const [selectedId, setSelectedId] = useState<string | null>(null);

  const recordById = useMemo(() => directoryRecordMap(rows, sourceQualityRowId), [rows]);
  const operateRows = useMemo(
    () => directoryOperateRows(rows, sourceQualityRowId, sourceQualityRowLabel),
    [rows]
  );
  const buildOverviewFields = useCallback(
    (row: SourceQualityRow) => buildSourceQualityOverviewFields(row, detailMode),
    [detailMode]
  );
  const nameColumnLabel = sourceQualityNameColumnLabel(draftGroupBy);

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={3} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load source quality" message={error.message} />;
  }

  const customerRequired = !draftCustomerId.trim();

  return (
    <PageLayout
      badge={
        detailMode ? (
          <Badge variant="outline">Geo/device detail</Badge>
        ) : freshness?.stale ? (
          <Badge variant="secondary">Stale data</Badge>
        ) : freshness?.as_of ? (
          <span className={adminTypography.bodyMuted}>As of {freshness.as_of}</span>
        ) : null
      }
      controlPanel={
        <DirectoryStack>
          <MetaLinksBand>
            <Link to="/reports">Reports catalog</Link>
          </MetaLinksBand>
          <FilterPanel>
            <DirectoryFilterForm layout="auto-fill" onSubmit={onApplyFilters}>
              <FilterField htmlFor="source-quality-customer" label="Customer" wide>
                <CustomerCombobox
                  id="source-quality-customer"
                  options={customerOptions}
                  value={draftCustomerId}
                  onValueChange={onDraftCustomerIdChange}
                />
              </FilterField>
              <FilterField htmlFor="source-quality-from" label="From">
                <DatetimePicker
                  id="source-quality-from"
                  label="From"
                  value={draftFrom}
                  onChange={onDraftFromChange}
                />
              </FilterField>
              <FilterField htmlFor="source-quality-to" label="To">
                <DatetimePicker
                  id="source-quality-to"
                  label="To"
                  value={draftTo}
                  onChange={onDraftToChange}
                />
              </FilterField>
              <FilterField htmlFor="source-quality-campaign" label="Campaign ID">
                <Input
                  id="source-quality-campaign"
                  value={draftCampaignId}
                  onChange={(event) => onDraftCampaignIdChange(event.target.value)}
                />
              </FilterField>
              <FilterField label="Group by" wide>
                <div className={cn('flex flex-wrap', adminSpacing.gap.lg)}>
                  {GROUP_BY_OPTIONS.map((option) => (
                    <div key={option.id} className={cn('flex items-center', adminSpacing.gap.md)}>
                      <Checkbox
                        checked={draftGroupBy.includes(option.id)}
                        id={`source-quality-group-${option.id}`}
                        onCheckedChange={(checked) => onToggleGroupBy(option.id, checked === true)}
                      />
                      <Label htmlFor={`source-quality-group-${option.id}`}>{option.label}</Label>
                    </div>
                  ))}
                </div>
              </FilterField>
              <FilterField label="Compare">
                <div className={cn('flex items-center', adminSpacing.gap.md, adminKit.controlHeight)}>
                  <Checkbox
                    checked={draftCompare}
                    id="source-quality-compare"
                    onCheckedChange={(checked) => onDraftCompareChange(checked === true)}
                  />
                  <Label htmlFor="source-quality-compare">Prior window</Label>
                </div>
              </FilterField>
              <FilterApplyButton disabled={fetching} type="submit" />
            </DirectoryFilterForm>
          </FilterPanel>
        </DirectoryStack>
      }
      description="Placement and sub-source quality with optional geo/device breakdown."
      title="Source quality"
      footer={
        rows.length > 0 ? (
          <DirectoryPaginationFooter
            canGoNext={canGoNext}
            canGoPrev={canGoPrev}
            disabled={fetching}
            limit={limit}
            pageSizeId="source-quality-page-size"
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
        <EmptyState title="No rows" description="No source quality rows for the current filters." />
      ) : (
        <TableHost className="w-full">
          <DirectorySelectOverviewTable
            buildOverviewFields={buildOverviewFields}
            disabled={fetching}
            nameColumnLabel={nameColumnLabel}
            overviewTitle={(row) => sourceQualityRowLabel(row)}
            recordById={recordById}
            revalidating={listRevalidating}
            rows={operateRows}
            selectedId={selectedId}
            onSelectedIdChange={setSelectedId}
          />
        </TableHost>
      )}
    </PageLayout>
  );
}
