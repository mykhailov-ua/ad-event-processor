import { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';

import type { PostbackReconRow } from '@/api/types';
import { FilterApplyButton } from '@/shell/action_buttons';
import { CustomerCombobox } from '@/shell/customer_combobox';
import { DirectoryFilterForm, FilterField, FilterPanel } from '@/shell/filter_panel';
import { PageLayout } from '@/shell/page_layout';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import { Badge } from '@/components/ui/badge';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { Input } from '@/components/ui/input';
import type { DirectoryOverviewField } from '@/shell/directory_overview_dialog';
import {
  DirectorySelectOverviewTable,
  directoryOperateRows,
  directoryRecordMap,
} from '@/shell/directory_select_overview_table';
import { displayMicro, displayTimestamp } from '@/lib/display';
import { adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import { DirectoryStack, MetaLinksBand, TableHost } from '@/shell/ui_bands';
import { usePostbackReconciliationPageWorkspace } from '@/domains/reports/use_postback_reconciliation_page_workspace';

function formatMicros(value?: number): string {
  if (value == null || !Number.isFinite(value)) {
    return '-';
  }
  return displayMicro(value) || String(value);
}

function postbackReconRowId(row: PostbackReconRow): string {
  const parts = [row.campaign_id, row.click_id, row.conversion_at].filter(Boolean);
  if (parts.length > 0) {
    return parts.join('|');
  }
  return '';
}

function postbackReconRowLabel(row: PostbackReconRow): string {
  return row.click_id ?? row.campaign_id ?? 'Conversion';
}

function buildPostbackReconOverviewFields(row: PostbackReconRow): DirectoryOverviewField[] {
  return [
    {
      label: 'Campaign',
      value: (
        <span className={cn(adminTypography.monoData, 'text-muted-foreground')}>
          {row.campaign_id ?? '-'}
        </span>
      ),
    },
    {
      label: 'Click ID',
      value: (
        <span className={cn(adminTypography.monoData, 'text-muted-foreground')}>
          {row.click_id ?? '-'}
        </span>
      ),
    },
    {
      label: 'Conversion at',
      value: row.conversion_at ? displayTimestamp(row.conversion_at) : '-',
    },
    { label: 'Conversion value', value: formatMicros(row.conversion_value_micro) },
    { label: 'Ledger fee', value: formatMicros(row.ledger_day_fee_micro) },
    { label: 'Postback', value: row.postback_status ?? '-' },
    {
      label: 'Reconcile',
      value: row.reconcile_status ? <Badge variant="outline">{row.reconcile_status}</Badge> : '-',
    },
    { label: 'Error', value: row.error_message ?? '-' },
  ];
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

  const [selectedId, setSelectedId] = useState<string | null>(null);

  const recordById = useMemo(() => directoryRecordMap(rows, postbackReconRowId), [rows]);
  const operateRows = useMemo(
    () => directoryOperateRows(rows, postbackReconRowId, postbackReconRowLabel),
    [rows]
  );

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={3} />;
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
          <span className={adminTypography.bodyMuted}>
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
          <DirectorySelectOverviewTable
            buildOverviewFields={buildPostbackReconOverviewFields}
            disabled={fetching}
            nameColumnLabel="Click ID"
            overviewTitle={(row) => postbackReconRowLabel(row)}
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
