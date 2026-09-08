import { useCallback, useEffect, useMemo, useState } from 'react';

import { getEdgeParityReport } from '@/api/reports_api';
import { useResource } from '@/api/use_resource';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { ReportKpiGrid } from '@/domains/reports/report_kpi_grid';
import { formatRatio } from '@/domains/reports/report_metric_display';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import { displayCount, displayTimestamp } from '@/lib/display';
import { FilterApplyButton } from '@/shell/action_buttons';
import { DirectoryFilterForm, FilterField, FilterPanel } from '@/shell/filter_panel';
import { PageLayout } from '@/shell/page_layout';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { DirectoryStack, MetaLinksBand, TableHost } from '@/shell/ui_bands';
import { Link } from 'react-router-dom';

function defaultEdgeParityRange(): { from: string; to: string } {
  const to = new Date();
  const from = new Date(to.getTime() - 15 * 60 * 1000);
  return { from: from.toISOString(), to: to.toISOString() };
}

export function EdgeParityPage() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const defaultRange = useMemo(() => defaultEdgeParityRange(), []);

  const appliedFrom = searchParams.get('from') ?? defaultRange.from;
  const appliedTo = searchParams.get('to') ?? defaultRange.to;

  const [draftFrom, setDraftFrom] = useState(toDatetimeLocalValue(appliedFrom));
  const [draftTo, setDraftTo] = useState(toDatetimeLocalValue(appliedTo));

  useEffect(() => {
    setDraftFrom(toDatetimeLocalValue(appliedFrom));
    setDraftTo(toDatetimeLocalValue(appliedTo));
  }, [appliedFrom, appliedTo]);

  const shouldFetch = Boolean(appliedFrom && appliedTo);

  const { data, error, fetching, revalidating } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return getEdgeParityReport({ from: appliedFrom, to: appliedTo }, signal);
    },
    [appliedFrom, appliedTo, shouldFetch]
  );

  const onApplyFilters = useCallback(
    (event?: { preventDefault?: () => void }) => {
      event?.preventDefault?.();
      const next = new URLSearchParams();
      const from = fromDatetimeLocalValue(draftFrom);
      const to = fromDatetimeLocalValue(draftTo);
      if (from) {
        next.set('from', from);
      }
      if (to) {
        next.set('to', to);
      }
      replaceSearchParams(next);
    },
    [draftFrom, draftTo, replaceSearchParams]
  );

  if ((fetching || listQueryPending) && !data && !error) {
    return <PageSkeleton variant="directory" columns={4} />;
  }

  if (error && !data) {
    return <ErrorBlock title="Could not load edge parity" message={error.message} />;
  }

  const badge = data?.alert ? (
    <Badge variant="destructive">Divergence alert</Badge>
  ) : data?.freshness?.stale ? (
    <Badge variant="secondary">Stale data</Badge>
  ) : data?.freshness?.as_of ? (
    <span className="text-xs text-muted-foreground">
      As of {displayTimestamp(data.freshness.as_of)}
    </span>
  ) : null;

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
              <FilterField htmlFor="edge-parity-from" label="From">
                <DatetimePicker
                  id="edge-parity-from"
                  label="From"
                  value={draftFrom}
                  onChange={setDraftFrom}
                />
              </FilterField>
              <FilterField htmlFor="edge-parity-to" label="To">
                <DatetimePicker
                  id="edge-parity-to"
                  label="To"
                  value={draftTo}
                  onChange={setDraftTo}
                />
              </FilterField>
              <FilterApplyButton disabled={fetching} type="submit" />
            </DirectoryFilterForm>
          </FilterPanel>
        </DirectoryStack>
      }
      description="Edge ingress vs tracker event parity (max 15 minute window)."
      title="Edge parity"
    >
      {!shouldFetch ? (
        <EmptyState title="Date range required" description="Set from and to, then apply." />
      ) : !data ? (
        <EmptyState title="No data" description="No parity snapshot for this window." />
      ) : (
        <TableHost className="w-full">
          <ReportKpiGrid
            items={[
              { label: 'Window from', value: displayTimestamp(data.from) || '-' },
              { label: 'Window to', value: displayTimestamp(data.to) || '-' },
              { label: 'Edge ingress', value: displayCount(data.edge_ingress) || '-' },
              { label: 'Tracker events', value: displayCount(data.tracker_events) || '-' },
              { label: 'Divergence', value: formatRatio(data.divergence_pct) },
              { label: 'Edge blocked', value: displayCount(data.edge_blocked_total) || '-' },
              { label: 'Blacklist stale', value: displayCount(data.blacklist_stale) || '-' },
              {
                label: 'Shard hint',
                value: data.shard_mismatch_hint?.trim() ? data.shard_mismatch_hint : '-',
              },
            ]}
          />
        </TableHost>
      )}
    </PageLayout>
  );
}
