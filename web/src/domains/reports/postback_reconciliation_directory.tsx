import { Link } from 'react-router-dom';

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
import { DirectoryFilterForm, FilterField, FilterPanel } from '@/shell/filter_panel';
import { PageLayout } from '@/shell/page_layout';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import { Badge } from '@/components/ui/badge';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { Input } from '@/components/ui/input';
import { displayCount, displayMicro, displayTimestamp } from '@/lib/display';
import { DirectoryStack, MetaLinksBand, TableHost } from '@/shell/ui_bands';
import { cn } from '@/lib/utils';
import { usePostbackReconciliationPageWorkspace } from '@/domains/reports/use_postback_reconciliation_page_workspace';

function formatMicros(value?: number): string {
  if (value == null || !Number.isFinite(value)) {
    return '-';
  }
  return displayMicro(value) || String(value);
}

export function PostbackReconciliationDirectory() {
  const {
    rows,
    freshness,
    customerOptions,
    draftCustomerId,
    draftFrom,
    draftTo,
    draftCampaignId,
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
    onApplyFilters,
    onPageChange,
  } = usePostbackReconciliationPageWorkspace();

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={7} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load postback reconciliation" message={error.message} />;
  }

  const customerRequired = !draftCustomerId.trim();

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
              <FilterField htmlFor="postback-recon-customer" label="Customer" wide>
                <CustomerCombobox
                  id="postback-recon-customer"
                  options={customerOptions}
                  value={draftCustomerId}
                  onValueChange={onDraftCustomerIdChange}
                />
              </FilterField>
              <FilterField htmlFor="postback-recon-from" label="From">
                <DatetimePicker
                  id="postback-recon-from"
                  label="From"
                  value={draftFrom}
                  onChange={onDraftFromChange}
                />
              </FilterField>
              <FilterField htmlFor="postback-recon-to" label="To">
                <DatetimePicker
                  id="postback-recon-to"
                  label="To"
                  value={draftTo}
                  onChange={onDraftToChange}
                />
              </FilterField>
              <FilterField htmlFor="postback-recon-campaign" label="Campaign ID">
                <Input
                  id="postback-recon-campaign"
                  value={draftCampaignId}
                  onChange={(event) => onDraftCampaignIdChange(event.target.value)}
                />
              </FilterField>
              <FilterApplyButton disabled={fetching} type="submit" />
            </DirectoryFilterForm>
          </FilterPanel>
        </DirectoryStack>
      }
      description="Postback dispatch status vs ledger reconciliation by conversion."
      title="Postback reconciliation"
      footer={
        rows.length > 0 ? (
          <DirectoryPaginationFooter
            canGoNext={canGoNext}
            canGoPrev={canGoPrev}
            disabled={fetching}
            limit={limit}
            pageSizeId="postback-recon-page-size"
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
        <EmptyState title="No rows" description="No reconciliation rows for the current filters." />
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
                <DirectoryTableHead>Click ID</DirectoryTableHead>
                <DirectoryTableHead>Conversion at</DirectoryTableHead>
                <DirectoryTableHead align="end">Conv. value</DirectoryTableHead>
                <DirectoryTableHead align="end">Ledger fee</DirectoryTableHead>
                <DirectoryTableHead>Postback</DirectoryTableHead>
                <DirectoryTableHead>Reconcile</DirectoryTableHead>
                <DirectoryTableHead>Error</DirectoryTableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row, index) => (
                <TableRow key={`${row.click_id ?? 'row'}-${index}`}>
                  <TableCell className="text-xs">{row.campaign_id ?? '-'}</TableCell>
                  <TableCell className="text-xs">{row.click_id ?? '-'}</TableCell>
                  <TableCell>
                    {row.conversion_at ? displayTimestamp(row.conversion_at) : '-'}
                  </TableCell>
                  <TableCell className="text-right">
                    {formatMicros(row.conversion_value_micro)}
                  </TableCell>
                  <TableCell className="text-right">
                    {formatMicros(row.ledger_day_fee_micro)}
                  </TableCell>
                  <TableCell>{row.postback_status ?? '-'}</TableCell>
                  <TableCell>
                    {row.reconcile_status ? (
                      <Badge variant="outline">{row.reconcile_status}</Badge>
                    ) : (
                      '-'
                    )}
                  </TableCell>
                  <TableCell className="max-w-md break-all text-sm">
                    {row.error_message ?? '-'}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </DirectoryTable>
        </TableHost>
      )}
    </PageLayout>
  );
}
