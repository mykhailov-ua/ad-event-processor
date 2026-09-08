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
import { Button } from '@/components/ui/button';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { Input } from '@/components/ui/input';
import type { DataFreshness, FraudReasonRow, FraudReasonsReportKey } from '@/api/types';
import { fraudReasonPlacementId, fraudReasonSignalsDegraded } from '@/api/types';
import {
  campaignListCellContentClass,
  campaignListCellContentNumClass,
  campaignListEllipsisTextClass,
  campaignListTableFullWidthClass,
  campaignListTdClass,
} from '@/domains/campaigns/list/campaign_list_classes';
import { displayCount, displayTimestamp } from '@/lib/display';
import { DirectoryStack, MetaLinksBand, TableHost } from '@/shell/ui_bands';
import { cn } from '@/lib/utils';

function formatSilentRejectRatio(value?: number): string {
  if (value == null || !Number.isFinite(value)) {
    return '';
  }
  return `${(value * 100).toFixed(1)}%`;
}

function reasonLabel(row: FraudReasonRow): string {
  return row.fraud_reason?.trim() || row.fraud_category_label?.trim() || row.fraud_category?.trim() || '-';
}

export type FraudReasonsDirectoryProps = {
  reportKey: FraudReasonsReportKey;
  title: string;
  description: string;
  rows: FraudReasonRow[];
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
  exporting: boolean;
  exportError: Error | undefined;
  exportTruncated: boolean;
  canGoPrev: boolean;
  canGoNext: boolean;
  limit: number;
  offset: number;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onDraftCampaignIdChange: (value: string) => void;
  onApplyFilters: (event?: { preventDefault?: () => void }) => void;
  onPageChange: (nextOffset: number) => void;
  onExportCsv: () => void;
};

export function FraudReasonsDirectory({
  reportKey,
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
  exporting,
  exportError,
  exportTruncated,
  canGoPrev,
  canGoNext,
  limit,
  offset,
  onDraftCustomerIdChange,
  onDraftFromChange,
  onDraftToChange,
  onDraftCampaignIdChange,
  onApplyFilters,
  onPageChange,
  onExportCsv,
}: FraudReasonsDirectoryProps) {
  const showPlacement = reportKey === 'fraud-breakdown';
  const showDegraded = reportKey === 'wire-signal-breakdown';

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={showPlacement ? 7 : 6} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title={`Could not load ${title.toLowerCase()}`} message={error.message} />;
  }

  return (
    <PageLayout
      badge={
        freshness?.stale ? (
          <Badge variant="secondary">stale CH lag {freshness.ch_lag_seconds ?? '?'}s</Badge>
        ) : freshness ? (
          <Badge variant="outline">{freshness.consistency ?? 'fresh'}</Badge>
        ) : undefined
      }
      controlPanel={
        <DirectoryStack>
          <MetaLinksBand>
            <Link to="/reports">Back to catalog</Link>
            <span>{description}</span>
            {freshness?.as_of ? (
              <span>As of {displayTimestamp(freshness.as_of, freshness.as_of_display)}</span>
            ) : null}
          </MetaLinksBand>
          <FilterPanel>
            <DirectoryFilterForm onSubmit={onApplyFilters}>
              <FilterField htmlFor="fraud-reasons-customer" label="Customer">
                <CustomerCombobox
                  id="fraud-reasons-customer"
                  disabled={fetching}
                  options={customerOptions}
                  value={draftCustomerId}
                  onValueChange={onDraftCustomerIdChange}
                />
              </FilterField>
              <DatetimePicker
                id="fraud-reasons-from"
                label="From"
                value={draftFrom}
                onChange={onDraftFromChange}
              />
              <DatetimePicker
                id="fraud-reasons-to"
                label="To"
                value={draftTo}
                onChange={onDraftToChange}
              />
              <FilterField htmlFor="fraud-reasons-campaign" label="Campaign ID">
                <Input
                  id="fraud-reasons-campaign"
                  placeholder="Optional"
                  value={draftCampaignId}
                  onChange={(event) => onDraftCampaignIdChange(event.target.value)}
                />
              </FilterField>
              <FilterApplyButton disabled={fetching || !draftCustomerId.trim()} type="submit">
                Run report
              </FilterApplyButton>
            </DirectoryFilterForm>
          </FilterPanel>
        </DirectoryStack>
      }
      footer={
        <div className="flex flex-wrap items-center gap-3">
          <DirectoryPaginationFooter
            canGoNext={canGoNext}
            canGoPrev={canGoPrev}
            disabled={fetching}
            variant="outline"
            onNext={() => onPageChange(offset + limit)}
            onPrev={() => onPageChange(Math.max(0, offset - limit))}
          />
          <Button
            disabled={fetching || rows.length === 0 || exporting}
            loading={exporting}
            type="button"
            variant="outline"
            onClick={onExportCsv}
          >
            Export CSV
          </Button>
        </div>
      }
      title={title}
    >
      {exportError ? <ErrorBlock title="Export failed" message={exportError.message} /> : null}
      {exportTruncated ? (
        <p className="text-sm text-muted-foreground" role="status">
          Export capped at 5,000 rows. Narrow filters or use report jobs for larger extracts.
        </p>
      ) : null}

      {rows.length === 0 ? (
        <EmptyState
          title="No fraud reasons"
          description="Pick a customer, set the date range, and run the report."
        />
      ) : (
        <TableHost className="w-full">
          <DirectoryTable
            className={cn(
              'w-full rounded-none border-0 shadow-none',
              directoryTableRevalidatingClass(listRevalidating)
            )}
            fixedLayout
            horizontalScroll
            nested
            tableClassName={campaignListTableFullWidthClass}
            tableStyle={{ width: '100%', tableLayout: 'fixed' }}
          >
            <colgroup>
              <col style={{ width: showPlacement ? '14%' : '16%' }} />
              <col style={{ width: showPlacement ? '22%' : '26%' }} />
              <col style={{ width: showPlacement ? '16%' : '18%' }} />
              {showPlacement ? <col style={{ width: '14%' }} /> : null}
              <col style={{ width: '10%' }} />
              <col style={{ width: '12%' }} />
              <col style={{ width: '10%' }} />
              {showDegraded ? <col style={{ width: '10%' }} /> : null}
            </colgroup>
            <TableHeader>
              <TableRow>
                <DirectoryTableHead>Campaign</DirectoryTableHead>
                <DirectoryTableHead>Reason</DirectoryTableHead>
                <DirectoryTableHead>Category</DirectoryTableHead>
                {showPlacement ? <DirectoryTableHead>Placement</DirectoryTableHead> : null}
                <DirectoryTableHead align="end">Events</DirectoryTableHead>
                <DirectoryTableHead align="end">Non-blocking</DirectoryTableHead>
                <DirectoryTableHead align="end">Ratio</DirectoryTableHead>
                {showDegraded ? <DirectoryTableHead>Degraded</DirectoryTableHead> : null}
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row, index) => {
                const rowKey = `${row.campaign_id ?? 'row'}-${row.fraud_reason ?? row.fraud_category ?? index}`;
                const label = reasonLabel(row);
                return (
                  <TableRow key={rowKey}>
                    <TableCell className={campaignListTdClass}>
                      {row.campaign_id ? (
                        <Link
                          className={cn(campaignListCellContentClass, 'text-primary hover:underline')}
                          to={`/campaigns/${row.campaign_id}/edit`}
                        >
                          {row.campaign_id}
                        </Link>
                      ) : (
                        <span className={campaignListCellContentClass}>-</span>
                      )}
                    </TableCell>
                    <TableCell className={campaignListTdClass}>
                      <span className={campaignListEllipsisTextClass} title={label}>
                        {label}
                      </span>
                    </TableCell>
                    <TableCell className={campaignListTdClass}>
                      <span
                        className={campaignListEllipsisTextClass}
                        title={row.fraud_category_label ?? row.fraud_category}
                      >
                        {row.fraud_category_label ?? row.fraud_category ?? '-'}
                      </span>
                    </TableCell>
                    {showPlacement ? (
                      <TableCell className={campaignListTdClass}>
                        <span
                          className={campaignListEllipsisTextClass}
                          title={fraudReasonPlacementId(row)}
                        >
                          {fraudReasonPlacementId(row) ?? '-'}
                        </span>
                      </TableCell>
                    ) : null}
                    <TableCell className={cn(campaignListTdClass, 'text-right')}>
                      <span className={campaignListCellContentNumClass}>
                        {displayCount(row.event_count)}
                      </span>
                    </TableCell>
                    <TableCell className={cn(campaignListTdClass, 'text-right')}>
                      <span className={campaignListCellContentNumClass}>
                        {displayCount(row.silent_reject_count)}
                      </span>
                    </TableCell>
                    <TableCell className={cn(campaignListTdClass, 'text-right')}>
                      <span className={campaignListCellContentNumClass}>
                        {formatSilentRejectRatio(row.silent_reject_ratio)}
                      </span>
                    </TableCell>
                    {showDegraded ? (
                      <TableCell className={campaignListTdClass}>
                        {fraudReasonSignalsDegraded(row) ? (
                          <Badge variant="secondary">degraded</Badge>
                        ) : (
                          <span className="text-muted-foreground">-</span>
                        )}
                      </TableCell>
                    ) : null}
                  </TableRow>
                );
              })}
            </TableBody>
          </DirectoryTable>
        </TableHost>
      )}

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageLayout>
  );
}
