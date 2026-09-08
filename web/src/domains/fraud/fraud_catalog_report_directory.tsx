import { Link } from 'react-router-dom';

import { FilterApplyButton } from '@/shell/action_buttons';
import { CustomerCombobox, type CustomerComboboxOption } from '@/shell/customer_combobox';
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { DataFreshness, FraudCatalogDimension, FraudCatalogReportRow } from '@/api/types';
import { renderFraudCatalogCell } from '@/domains/fraud/fraud_catalog_report_cells';
import type { FraudCatalogReportMeta } from '@/domains/fraud/fraud_catalog_report_meta';
import { displayTimestamp } from '@/lib/display';
import { DirectoryStack, MetaLinksBand, TableHost } from '@/shell/ui_bands';
import { cn } from '@/lib/utils';

export type FraudCatalogReportDirectoryProps = {
  meta: FraudCatalogReportMeta;
  rows: FraudCatalogReportRow[];
  seriesRows: FraudCatalogReportRow[];
  truncated: boolean;
  freshness?: DataFreshness;
  customerOptions: CustomerComboboxOption[];
  draftCustomerId: string;
  draftFrom: string;
  draftTo: string;
  draftCampaignId: string;
  draftDimension: FraudCatalogDimension;
  draftCompare: boolean;
  draftSlice: boolean;
  draftMinDesync: string;
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
  onDraftDimensionChange: (value: FraudCatalogDimension) => void;
  onDraftCompareChange: (value: boolean) => void;
  onDraftSliceChange: (value: boolean) => void;
  onDraftMinDesyncChange: (value: string) => void;
  onApplyFilters: (event?: { preventDefault?: () => void }) => void;
  onPageChange: (nextOffset: number) => void;
};

export function FraudCatalogReportDirectory({
  meta,
  rows,
  seriesRows,
  truncated,
  freshness,
  customerOptions,
  draftCustomerId,
  draftFrom,
  draftTo,
  draftCampaignId,
  draftDimension,
  draftCompare,
  draftSlice,
  draftMinDesync,
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
  onDraftDimensionChange,
  onDraftCompareChange,
  onDraftSliceChange,
  onDraftMinDesyncChange,
  onApplyFilters,
  onPageChange,
}: FraudCatalogReportDirectoryProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={meta.columns.length} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title={`Could not load ${meta.title}`} message={error.message} />;
  }

  const customerRequired = meta.requiresCustomer && !draftCustomerId.trim();

  return (
    <PageLayout
      badge={
        freshness?.stale ? (
          <Badge variant="secondary">Stale data</Badge>
        ) : freshness?.as_of ? (
          <span className="text-xs text-muted-foreground">
            As of {displayTimestamp(freshness.as_of)}
          </span>
        ) : truncated ? (
          <Badge variant="secondary">Truncated</Badge>
        ) : null
      }
      controlPanel={
        <DirectoryStack>
          <MetaLinksBand>
            <Link to="/fraud">Fraud hub</Link>
            <Link to="/reports">Reports catalog</Link>
          </MetaLinksBand>
          <FilterPanel>
            <DirectoryFilterForm layout="auto-fill" onSubmit={onApplyFilters}>
              {meta.requiresCustomer ? (
                <FilterField htmlFor={`${meta.key}-customer`} label="Customer" wide>
                  <CustomerCombobox
                    id={`${meta.key}-customer`}
                    options={customerOptions}
                    value={draftCustomerId}
                    onValueChange={onDraftCustomerIdChange}
                  />
                </FilterField>
              ) : null}
              <FilterField htmlFor={`${meta.key}-from`} label="From">
                <DatetimePicker
                  id={`${meta.key}-from`}
                  label="From"
                  value={draftFrom}
                  onChange={onDraftFromChange}
                />
              </FilterField>
              <FilterField htmlFor={`${meta.key}-to`} label="To">
                <DatetimePicker
                  id={`${meta.key}-to`}
                  label="To"
                  value={draftTo}
                  onChange={onDraftToChange}
                />
              </FilterField>
              {meta.requiresCustomer ? (
                <FilterField htmlFor={`${meta.key}-campaign`} label="Campaign ID">
                  <Input
                    id={`${meta.key}-campaign`}
                    value={draftCampaignId}
                    onChange={(event) => onDraftCampaignIdChange(event.target.value)}
                  />
                </FilterField>
              ) : null}
              {meta.showDimensionFilter ? (
                <FilterField htmlFor={`${meta.key}-dimension`} label="Dimension">
                  <Select
                    value={draftDimension}
                    onValueChange={(value) =>
                      onDraftDimensionChange(value as FraudCatalogDimension)
                    }
                  >
                    <SelectTrigger id={`${meta.key}-dimension`} className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="placement">Placement</SelectItem>
                      <SelectItem value="sub1">Sub1</SelectItem>
                      <SelectItem value="sub2">Sub2</SelectItem>
                      <SelectItem value="country">Country</SelectItem>
                      <SelectItem value="campaign">Campaign</SelectItem>
                    </SelectContent>
                  </Select>
                </FilterField>
              ) : null}
              {meta.showCompareFilter ? (
                <div className="flex items-center gap-2 self-end">
                  <Checkbox
                    checked={draftCompare}
                    id={`${meta.key}-compare`}
                    onCheckedChange={(checked) => onDraftCompareChange(checked === true)}
                  />
                  <Label htmlFor={`${meta.key}-compare`}>Compare prior window</Label>
                </div>
              ) : null}
              {meta.showMinDesyncFilter ? (
                <FilterField htmlFor={`${meta.key}-min-desync`} label="Min desync layers">
                  <Input
                    id={`${meta.key}-min-desync`}
                    inputMode="numeric"
                    min={1}
                    max={255}
                    value={draftMinDesync}
                    onChange={(event) => onDraftMinDesyncChange(event.target.value)}
                  />
                </FilterField>
              ) : null}
              {meta.showSliceFilter ? (
                <div className="flex items-center gap-2 self-end">
                  <Checkbox
                    checked={draftSlice}
                    id={`${meta.key}-slice`}
                    onCheckedChange={(checked) => onDraftSliceChange(checked === true)}
                  />
                  <Label htmlFor={`${meta.key}-slice`}>Slice by placement and geo</Label>
                </div>
              ) : null}
              <FilterApplyButton disabled={fetching} type="submit" />
            </DirectoryFilterForm>
          </FilterPanel>
        </DirectoryStack>
      }
      description={meta.description}
      footer={
        rows.length > 0 ? (
          <DirectoryPaginationFooter
            canGoNext={canGoNext}
            canGoPrev={canGoPrev}
            disabled={fetching}
            limit={limit}
            pageSizeId={`${meta.key}-page-size`}
            rangeLabel={rangeLabel}
            showPrevNext
            onNext={() => onPageChange(offset + limit)}
            onPrev={() => onPageChange(Math.max(0, offset - limit))}
          />
        ) : null
      }
      title={meta.title}
    >
      {customerRequired ? (
        <EmptyState
          title="Customer required"
          description="Pick a customer and date range, then apply filters."
        />
      ) : rows.length === 0 ? (
        <EmptyState title="No rows" description="No data for the current filters." />
      ) : (
        <TableHost className="w-full">
          <DirectoryTable
            className={cn('w-full', directoryTableRevalidatingClass(listRevalidating))}
            horizontalScroll
            nested
          >
            <TableHeader>
              <TableRow>
                {meta.columns.map((column) => (
                  <DirectoryTableHead key={column.id} align={column.align === 'right' ? 'end' : undefined}>
                    {column.label}
                  </DirectoryTableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row, index) => (
                <TableRow key={`${meta.key}-row-${index}`}>
                  {meta.columns.map((column) => (
                    <TableCell key={column.id} className={column.align === 'right' ? 'text-right' : undefined}>
                      {renderFraudCatalogCell(row, column)}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </DirectoryTable>
        </TableHost>
      )}
      {seriesRows.length > 0 && meta.seriesColumns ? (
        <TableHost className="w-full">
          <h2 className="text-sm font-medium">Hourly series</h2>
          <DirectoryTable className="w-full" horizontalScroll nested>
            <TableHeader>
              <TableRow>
                {meta.seriesColumns.map((column) => (
                  <DirectoryTableHead key={column.id} align={column.align === 'right' ? 'end' : undefined}>
                    {column.label}
                  </DirectoryTableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {seriesRows.map((row, index) => (
                <TableRow key={`${meta.key}-series-${index}`}>
                  {meta.seriesColumns?.map((column) => (
                    <TableCell key={column.id} className={column.align === 'right' ? 'text-right' : undefined}>
                      {renderFraudCatalogCell(row, column)}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </DirectoryTable>
        </TableHost>
      ) : null}
    </PageLayout>
  );
}
