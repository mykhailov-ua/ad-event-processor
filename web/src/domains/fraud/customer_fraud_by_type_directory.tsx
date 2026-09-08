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
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { Input } from '@/components/ui/input';
import type { CustomerFraudByTypeRow, DataFreshness } from '@/api/types';
import { displayCount, displayTimestamp } from '@/lib/display';
import { DirectoryStack, MetaLinksBand, TableHost } from '@/shell/ui_bands';
import { cn } from '@/lib/utils';

function formatSharePct(value?: number): string {
  if (value == null || !Number.isFinite(value)) {
    return '';
  }
  return `${value.toFixed(1)}%`;
}

function formatRatio(value?: number): string {
  if (value == null || !Number.isFinite(value)) {
    return '';
  }
  return `${(value * 100).toFixed(1)}%`;
}

export type CustomerFraudByTypeDirectoryProps = {
  title: string;
  description: string;
  rows: CustomerFraudByTypeRow[];
  freshness?: DataFreshness;
  customerOptions: CustomerComboboxOption[];
  draftCustomerId: string;
  draftFrom: string;
  draftTo: string;
  draftCampaignId: string;
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
  onApplyFilters: (event?: { preventDefault?: () => void }) => void;
  onPageChange: (nextOffset: number) => void;
};

export function CustomerFraudByTypeDirectory({
  title,
  description,
  rows,
  freshness,
  customerOptions,
  draftCustomerId,
  draftFrom,
  draftTo,
  draftCampaignId,
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
  onApplyFilters,
  onPageChange,
}: CustomerFraudByTypeDirectoryProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={6} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load fraud by type" message={error.message} />;
  }

  return (
    <PageLayout
      badge={
        freshness?.stale ? (
          <Badge variant="secondary">Stale data</Badge>
        ) : freshness?.as_of ? (
          <span className="text-xs text-muted-foreground">
            As of {displayTimestamp(freshness.as_of)}
          </span>
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
              <FilterField htmlFor="fraud-type-customer" label="Customer" wide>
                <CustomerCombobox
                  id="fraud-type-customer"
                  options={customerOptions}
                  value={draftCustomerId}
                  onValueChange={onDraftCustomerIdChange}
                />
              </FilterField>
              <FilterField htmlFor="fraud-type-from" label="From">
                <DatetimePicker
                  id="fraud-type-from"
                  label="From"
                  value={draftFrom}
                  onChange={onDraftFromChange}
                />
              </FilterField>
              <FilterField htmlFor="fraud-type-to" label="To">
                <DatetimePicker
                  id="fraud-type-to"
                  label="To"
                  value={draftTo}
                  onChange={onDraftToChange}
                />
              </FilterField>
              <FilterField htmlFor="fraud-type-campaign" label="Campaign ID">
                <Input
                  id="fraud-type-campaign"
                  value={draftCampaignId}
                  onChange={(event) => onDraftCampaignIdChange(event.target.value)}
                />
              </FilterField>
              <FilterApplyButton disabled={fetching} type="submit" />
            </DirectoryFilterForm>
          </FilterPanel>
        </DirectoryStack>
      }
      description={description}
      footer={
        rows.length > 0 ? (
          <DirectoryPaginationFooter
            canGoNext={canGoNext}
            canGoPrev={canGoPrev}
            disabled={fetching}
            limit={limit}
            pageSizeId="fraud-by-type-page-size"
            rangeLabel={rangeLabel}
            showPrevNext
            onNext={() => onPageChange(offset + limit)}
            onPrev={() => onPageChange(Math.max(0, offset - limit))}
          />
        ) : null
      }
      title={title}
    >
      {!draftCustomerId.trim() ? (
        <EmptyState
          title="Customer required"
          description="Pick a customer and date range, then apply filters."
        />
      ) : rows.length === 0 ? (
        <EmptyState title="No fraud by type" description="No rows for the current filters." />
      ) : (
        <TableHost className="w-full">
          <DirectoryTable
            className={cn('w-full', directoryTableRevalidatingClass(listRevalidating))}
            horizontalScroll
            nested
          >
            <TableHeader>
              <TableRow>
                <DirectoryTableHead>Campaign</DirectoryTableHead>
                <DirectoryTableHead>Category</DirectoryTableHead>
                <DirectoryTableHead align="end">Events</DirectoryTableHead>
                <DirectoryTableHead align="end">Non-blocking</DirectoryTableHead>
                <DirectoryTableHead align="end">Share</DirectoryTableHead>
                <DirectoryTableHead align="end">Non-blocking ratio</DirectoryTableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row, index) => (
                <TableRow key={`${row.campaign_id}-${row.fraud_category}-${index}`}>
                  <TableCell className="font-mono text-xs">{row.campaign_id ?? ''}</TableCell>
                  <TableCell>
                    {row.fraud_category_label?.trim() || row.fraud_category?.trim() || '-'}
                  </TableCell>
                  <TableCell align="right">{displayCount(row.event_count)}</TableCell>
                  <TableCell align="right">{displayCount(row.silent_reject_count)}</TableCell>
                  <TableCell align="right">
                    {row.share_label?.trim() || formatSharePct(row.share_pct)}
                  </TableCell>
                  <TableCell align="right">{formatRatio(row.silent_reject_ratio)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </DirectoryTable>
        </TableHost>
      )}
    </PageLayout>
  );
}
