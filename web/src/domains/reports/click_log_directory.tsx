import { Link } from 'react-router-dom';

import { FilterApplyButton } from '@/components/system/action_buttons';
import { CustomerCombobox, type CustomerComboboxOption } from '@/components/system/customer_combobox';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/components/system/directory_table';
import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
} from '@/components/system/filter_panel';
import { PageChrome } from '@/components/system/page_chrome';
import { EmptyState } from '@/components/system/empty_state';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { PaginationPrevNext } from '@/components/system/pagination_prev_next';
import { Badge } from '@/components/ui/badge';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { ClickLogEvent, ClickLogPostback, DataFreshness } from '@/api/types';
import { displayMicro, displayTimestamp } from '@/lib/display';

function buildClickLogTimelineHref(customerId: string, clickId: string): string {
  const params = new URLSearchParams();
  if (customerId.trim()) {
    params.set('customer_id', customerId.trim());
  }
  params.set('click_id', clickId);
  return `/reports/click-log?${params.toString()}`;
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
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load click log" message={error.message} />;
  }

  const canGoNext = Boolean(nextCursor) && !timelineMode;

  return (
    <PageChrome
      title="Click log"
      badge={
        freshness?.stale ? (
          <Badge variant="secondary">stale CH lag {freshness.ch_lag_seconds ?? '?'}s</Badge>
        ) : freshness ? (
          <Badge variant="outline">{freshness.consistency ?? 'fresh'}</Badge>
        ) : undefined
      }
    >
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
          <DatetimePicker id="click-log-to" label="To" value={draftTo} onChange={onDraftToChange} />
          <div className="grid gap-2">
            <Label htmlFor="click-log-campaign">Campaign ID</Label>
            <Input
              id="click-log-campaign"
              placeholder="Optional"
              value={draftCampaignId}
              onChange={(event) => onDraftCampaignIdChange(event.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="click-log-click-id">Click ID</Label>
            <Input
              id="click-log-click-id"
              placeholder="Timeline mode"
              value={draftClickId}
              onChange={(event) => onDraftClickIdChange(event.target.value)}
            />
          </div>
          <FilterApplyButton disabled={fetching || !draftCustomerId.trim()} type="submit">
            Apply
          </FilterApplyButton>
        </DirectoryFilterForm>
      </FilterPanel>

      {timelineMode ? (
        <p className="text-sm text-muted-foreground">
          Timeline for click <span className="font-mono text-foreground">{draftClickId.trim()}</span>
        </p>
      ) : null}

      {events.length === 0 ? (
        <EmptyState
          title="No click events"
          description="Adjust filters or pick a different date range."
        />
      ) : (
        <DirectoryTable scrollable>
          <TableHeader>
            <TableRow>
              <DirectoryTableHead className="w-[10%]">Type</DirectoryTableHead>
              <DirectoryTableHead className="w-[18%]">Click ID</DirectoryTableHead>
              <DirectoryTableHead className="w-[16%]">Time</DirectoryTableHead>
              <DirectoryTableHead className="w-[16%]">Campaign</DirectoryTableHead>
              <DirectoryTableHead className="w-[8%]">Country</DirectoryTableHead>
              <DirectoryTableHead className="w-[12%]">Source</DirectoryTableHead>
              <DirectoryTableHead align="end" className="w-[10%]">
                Cost
              </DirectoryTableHead>
              <DirectoryTableHead align="end" className="w-[10%]">
                Revenue
              </DirectoryTableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {events.map((event) => (
              <TableRow key={`${event.click_id}-${event.event_type}-${event.created_at}`}>
                <TableCell className="capitalize text-sm">{event.event_type ?? 'click'}</TableCell>
                <TableCell className="max-w-0 truncate font-mono text-xs">
                  {event.click_id ? (
                    <Link
                      className="text-primary hover:underline"
                      to={buildClickLogTimelineHref(draftCustomerId, event.click_id)}
                    >
                      {event.click_id}
                    </Link>
                  ) : (
                    '—'
                  )}
                </TableCell>
                <TableCell className="text-sm tabular-nums">
                  {displayTimestamp(event.created_at)}
                </TableCell>
                <TableCell className="max-w-0 truncate text-sm">
                  {event.campaign_id ? (
                    <Link
                      className="text-primary hover:underline"
                      to={`/campaigns/${event.campaign_id}/edit`}
                    >
                      {event.campaign_id}
                    </Link>
                  ) : (
                    '—'
                  )}
                </TableCell>
                <TableCell className="text-sm">{event.country ?? '—'}</TableCell>
                <TableCell className="max-w-0 truncate text-sm" title={event.sub1}>
                  {event.sub1 ?? '—'}
                </TableCell>
                <TableCell className="text-right text-sm tabular-nums">
                  {displayMicro(event.attributed_cost_micro)}
                </TableCell>
                <TableCell className="text-right text-sm tabular-nums">
                  {displayMicro(event.revenue_micro)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </DirectoryTable>
      )}

      {!timelineMode ? (
        <PaginationPrevNext
          canGoNext={canGoNext}
          canGoPrev={canGoPrev}
          onNext={onNextPage}
          onPrev={onPrevPage}
        />
      ) : null}

      {timelineMode && postbacks.length > 0 ? (
        <section className="ui-surface-raised grid gap-3 p-5">
          <h3 className="text-base font-medium tracking-tight">Postbacks</h3>
          <DirectoryTable className="rounded-none border-0 bg-transparent">
            <TableHeader>
              <TableRow>
                <DirectoryTableHead>Status</DirectoryTableHead>
                <DirectoryTableHead>Time</DirectoryTableHead>
                <DirectoryTableHead>Error</DirectoryTableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {postbacks.map((postback) => (
                <TableRow key={`${postback.status}-${postback.created_at}`}>
                  <TableCell>{postback.status}</TableCell>
                  <TableCell className="tabular-nums">
                    {displayTimestamp(postback.created_at)}
                  </TableCell>
                  <TableCell className="max-w-0 truncate text-muted-foreground">
                    {postback.error_message ?? '—'}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </DirectoryTable>
        </section>
      ) : null}

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageChrome>
  );
}
