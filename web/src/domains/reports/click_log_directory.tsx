import { useCallback, useMemo, useState } from 'react';
import type { ClickLogEvent, ClickLogPostback, DataFreshness } from '@/api/types';
import { FilterApplyButton } from '@/shell/action_buttons';
import { CustomerCombobox, type CustomerComboboxOption } from '@/shell/customer_combobox';
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
import { InAppLink } from '@/shell/in_app_link';
import { displayMicro, displayTimestamp } from '@/lib/display';
import { adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import { TableHost } from '@/shell/ui_bands';

function buildClickLogTimelineHref(customerId: string, clickId: string): string {
  const params = new URLSearchParams();
  if (customerId.trim()) {
    params.set('customer_id', customerId.trim());
  }
  params.set('click_id', clickId);
  return `/reports/click-log?${params.toString()}`;
}

function clickLogEventId(event: ClickLogEvent): string {
  const parts = [event.click_id, event.event_type, event.created_at].filter(Boolean);
  return parts.length > 0 ? parts.join('|') : '';
}

function clickLogEventLabel(event: ClickLogEvent): string {
  return event.click_id ?? event.event_type ?? 'Click event';
}

function postbackRowId(postback: ClickLogPostback): string {
  const parts = [postback.status, postback.created_at, postback.error_message].filter(Boolean);
  return parts.length > 0 ? parts.join('|') : '';
}

function postbackRowLabel(postback: ClickLogPostback): string {
  return postback.status ?? 'Postback';
}

function buildClickLogEventOverviewFields(
  event: ClickLogEvent,
  customerId: string
): DirectoryOverviewField[] {
  return [
    { label: 'Type', value: event.event_type ?? 'click' },
    {
      label: 'Click ID',
      value: event.click_id ? (
        <InAppLink
          className="text-primary hover:underline"
          to={buildClickLogTimelineHref(customerId, event.click_id)}
        >
          <span className={adminTypography.monoData}>{event.click_id}</span>
        </InAppLink>
      ) : (
        '-'
      ),
    },
    { label: 'Time', value: displayTimestamp(event.created_at) },
    {
      label: 'Campaign',
      value: event.campaign_id ? (
        <InAppLink className="text-primary hover:underline" to={`/campaigns/${event.campaign_id}/edit`}>
          <span className={adminTypography.monoData}>{event.campaign_id}</span>
        </InAppLink>
      ) : (
        '-'
      ),
    },
    { label: 'Country', value: event.country ?? '-' },
    { label: 'Source', value: event.sub1 ?? '-' },
    { label: 'Cost', value: displayMicro(event.attributed_cost_micro) },
    { label: 'Revenue', value: displayMicro(event.revenue_micro) },
  ];
}

function buildPostbackOverviewFields(postback: ClickLogPostback): DirectoryOverviewField[] {
  return [
    { label: 'Status', value: postback.status ?? '-' },
    { label: 'Time', value: displayTimestamp(postback.created_at) },
    { label: 'Error', value: postback.error_message ?? '-' },
  ];
}

export type ClickLogDirectoryProps = {
  events: ClickLogEvent[];
  postbacks: ClickLogPostback[];
  freshness?: DataFreshness;
  nextCursor?: string;
  timelineMode: boolean;
  customerOptions: CustomerComboboxOption[];
  draftCustomerId: string;
  draftFrom: string;
  draftTo: string;
  draftCampaignId: string;
  draftClickId: string;
  cursor?: string;
  fetching: boolean;
  listRevalidating?: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onDraftCampaignIdChange: (value: string) => void;
  onDraftClickIdChange: (value: string) => void;
  onApplyFilters: (event?: { preventDefault?: () => void }) => void;
  onNextPage: () => void;
  onPrevPage: () => void;
  canGoPrev: boolean;
};

export function ClickLogDirectory({
  events,
  postbacks,
  freshness,
  nextCursor,
  timelineMode,
  customerOptions,
  draftCustomerId,
  draftFrom,
  draftTo,
  draftCampaignId,
  draftClickId,
  fetching,
  listRevalidating = false,
  error,
  hasSnapshot,
  onDraftCustomerIdChange,
  onDraftFromChange,
  onDraftToChange,
  onDraftCampaignIdChange,
  onDraftClickIdChange,
  onApplyFilters,
  onNextPage,
  onPrevPage,
  canGoPrev,
}: ClickLogDirectoryProps) {
  const [selectedEventId, setSelectedEventId] = useState<string | null>(null);
  const [selectedPostbackId, setSelectedPostbackId] = useState<string | null>(null);

  const eventRecordById = useMemo(() => directoryRecordMap(events, clickLogEventId), [events]);
  const eventRows = useMemo(
    () => directoryOperateRows(events, clickLogEventId, clickLogEventLabel),
    [events]
  );
  const buildEventOverviewFields = useCallback(
    (event: ClickLogEvent) => buildClickLogEventOverviewFields(event, draftCustomerId),
    [draftCustomerId]
  );

  const postbackRecordById = useMemo(() => directoryRecordMap(postbacks, postbackRowId), [postbacks]);
  const postbackRows = useMemo(
    () => directoryOperateRows(postbacks, postbackRowId, postbackRowLabel),
    [postbacks]
  );

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load click log" message={error.message} />;
  }

  const canGoNext = Boolean(nextCursor) && !timelineMode;

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
        <FilterPanel>
          <DirectoryFilterForm onSubmit={onApplyFilters}>
            <FilterField htmlFor="click-log-customer" label="Customer">
              <CustomerCombobox
                id="click-log-customer"
                disabled={fetching}
                options={customerOptions}
                value={draftCustomerId}
                onValueChange={onDraftCustomerIdChange}
              />
            </FilterField>
            <DatetimePicker
              id="click-log-from"
              label="From"
              value={draftFrom}
              onChange={onDraftFromChange}
            />
            <DatetimePicker
              id="click-log-to"
              label="To"
              value={draftTo}
              onChange={onDraftToChange}
            />
            <FilterField htmlFor="click-log-campaign" label="Campaign ID">
              <Input
                id="click-log-campaign"
                placeholder="Optional"
                value={draftCampaignId}
                onChange={(event) => onDraftCampaignIdChange(event.target.value)}
              />
            </FilterField>
            <FilterField htmlFor="click-log-click-id" label="Click ID">
              <Input
                id="click-log-click-id"
                placeholder="Timeline mode"
                value={draftClickId}
                onChange={(event) => onDraftClickIdChange(event.target.value)}
              />
            </FilterField>
            <FilterApplyButton disabled={fetching || !draftCustomerId.trim()} type="submit">
              Apply
            </FilterApplyButton>
          </DirectoryFilterForm>
        </FilterPanel>
      }
      footer={
        !timelineMode ? (
          <DirectoryPaginationFooter
            canGoNext={canGoNext}
            canGoPrev={canGoPrev}
            variant="outline"
            onNext={onNextPage}
            onPrev={onPrevPage}
          />
        ) : null
      }
      title="Click log"
    >
      {timelineMode ? (
        <p className={adminTypography.bodyMuted}>
          Timeline for click{' '}
          <span className={cn(adminTypography.monoData, 'text-foreground')}>
            {draftClickId.trim()}
          </span>
        </p>
      ) : null}

      {events.length === 0 ? (
        <EmptyState
          title="No click events"
          description="Adjust filters or pick a different date range."
        />
      ) : (
        <TableHost className="w-full">
          <DirectorySelectOverviewTable
            buildOverviewFields={buildEventOverviewFields}
            disabled={fetching}
            nameColumnLabel="Click ID"
            overviewTitle={(event) => clickLogEventLabel(event)}
            recordById={eventRecordById}
            revalidating={listRevalidating}
            rows={eventRows}
            selectedId={selectedEventId}
            onSelectedIdChange={setSelectedEventId}
          />
        </TableHost>
      )}

      {timelineMode && postbacks.length > 0 ? (
        <section className={cn('grid', 'gap-2')}>
          <h3 className={adminTypography.sectionTitle}>Postbacks</h3>
          <TableHost className="w-full">
            <DirectorySelectOverviewTable
              buildOverviewFields={buildPostbackOverviewFields}
              disabled={fetching}
              nameColumnLabel="Status"
              overviewTitle={(postback) => postbackRowLabel(postback)}
              recordById={postbackRecordById}
              rows={postbackRows}
              selectedId={selectedPostbackId}
              onSelectedIdChange={setSelectedPostbackId}
            />
          </TableHost>
        </section>
      ) : null}

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageLayout>
  );
}
