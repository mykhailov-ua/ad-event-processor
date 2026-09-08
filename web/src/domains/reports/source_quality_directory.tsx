import { Link } from 'react-router-dom';

import type { ReportCompareDeltas, SourceQualityGroupBy, SourceQualityRow } from '@/api/types';
import { FilterApplyButton } from '@/shell/action_buttons';
import { CustomerCombobox } from '@/shell/customer_combobox';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
  directoryTableRevalidatingClass,
} from '@/shell/directory_table';
import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
} from '@/shell/filter_panel';
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
import { displayCount, displayMicro } from '@/lib/display';
import { DirectoryStack, MetaLinksBand, TableHost } from '@/shell/ui_bands';
import { cn } from '@/lib/utils';
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

function metricCell(value: number | undefined, delta?: number, format: 'count' | 'micro' = 'count') {
  const text =
    format === 'micro' ? displayMicro(value) || '-' : displayCount(value) || '-';
  const suffix = deltaSuffix(delta);
  if (!suffix) {
    return text;
  }
  return (
    <span>
      {text}
      <span className="ml-1 text-xs text-muted-foreground">{suffix}</span>
    </span>
  );
}

function SourceQualityRowCells({
  row,
  detailMode,
}: {
  row: SourceQualityRow;
  detailMode: boolean;
}) {
  const compare: ReportCompareDeltas | undefined = row.compare;
  return (
    <>
      <TableCell className="font-mono text-xs">{row.placement_id ?? '-'}</TableCell>
      <TableCell className="font-mono text-xs">{row.campaign_id ?? '-'}</TableCell>
      {detailMode ? (
        <>
          <TableCell>{row.country ?? '-'}</TableCell>
          <TableCell>{row.city ?? '-'}</TableCell>
          <TableCell>{row.device ?? '-'}</TableCell>
          <TableCell className="font-mono text-xs">{row.sub1 ?? '-'}</TableCell>
        </>
      ) : null}
      <TableCell className="text-right">
        {metricCell(row.impressions, compare?.impressions_delta)}
      </TableCell>
      <TableCell className="text-right">{metricCell(row.clicks, compare?.clicks_delta)}</TableCell>
      <TableCell className="text-right">
        {metricCell(row.conversions, compare?.conversions_delta)}
      </TableCell>
      <TableCell className="text-right">
        {metricCell(row.spend_micro, compare?.spend_micro_delta, 'micro')}
      </TableCell>
      <TableCell className="text-right">
        {metricCell(row.revenue_micro, compare?.revenue_micro_delta, 'micro')}
      </TableCell>
      <TableCell className="text-right">{displayMicro(row.profit_micro) || '-'}</TableCell>
      <TableCell className="text-right">{formatPct(row.roi_pct)}</TableCell>
      <TableCell className="text-right">{displayMicro(row.cpa_micro) || '-'}</TableCell>
      <TableCell className="text-right">{formatRatio(row.ctr)}</TableCell>
      <TableCell className="text-right">{formatRatio(row.ivt_rate)}</TableCell>
    </>
  );
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

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={8} />;
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
          <span className="text-xs text-muted-foreground">As of {freshness.as_of}</span>
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
                <div className="flex flex-wrap gap-3">
                  {GROUP_BY_OPTIONS.map((option) => (
                    <div key={option.id} className="flex items-center gap-2">
                      <Checkbox
                        checked={draftGroupBy.includes(option.id)}
                        id={`source-quality-group-${option.id}`}
                        onCheckedChange={(checked) =>
                          onToggleGroupBy(option.id, checked === true)
                        }
                      />
                      <Label htmlFor={`source-quality-group-${option.id}`}>{option.label}</Label>
                    </div>
                  ))}
                </div>
              </FilterField>
              <FilterField label="Compare">
                <div className="flex h-9 items-center gap-2">
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
          <DirectoryTable
            className={cn('w-full', directoryTableRevalidatingClass(listRevalidating))}
            horizontalScroll
            nested
          >
            <TableHeader>
              <TableRow>
                <DirectoryTableHead>Placement</DirectoryTableHead>
                <DirectoryTableHead>Campaign</DirectoryTableHead>
                {detailMode ? (
                  <>
                    <DirectoryTableHead>Country</DirectoryTableHead>
                    <DirectoryTableHead>City</DirectoryTableHead>
                    <DirectoryTableHead>Device</DirectoryTableHead>
                    <DirectoryTableHead>Sub</DirectoryTableHead>
                  </>
                ) : null}
                <DirectoryTableHead align="end">Impressions</DirectoryTableHead>
                <DirectoryTableHead align="end">Clicks</DirectoryTableHead>
                <DirectoryTableHead align="end">Conversions</DirectoryTableHead>
                <DirectoryTableHead align="end">Spend</DirectoryTableHead>
                <DirectoryTableHead align="end">Revenue</DirectoryTableHead>
                <DirectoryTableHead align="end">Profit</DirectoryTableHead>
                <DirectoryTableHead align="end">ROI</DirectoryTableHead>
                <DirectoryTableHead align="end">CPA</DirectoryTableHead>
                <DirectoryTableHead align="end">CTR</DirectoryTableHead>
                <DirectoryTableHead align="end">IVT</DirectoryTableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row, index) => (
                <TableRow key={`${row.placement_id ?? 'p'}-${row.campaign_id ?? 'c'}-${index}`}>
                  <SourceQualityRowCells detailMode={detailMode} row={row} />
                </TableRow>
              ))}
            </TableBody>
          </DirectoryTable>
        </TableHost>
      )}
    </PageLayout>
  );
}
