import { useMemo, useState } from 'react';

import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import type { DLQInboxEntry } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import { EmptyState } from '@/shell/empty_state';
import type { DirectoryOverviewField } from '@/shell/directory_overview_dialog';
import { DirectoryRowActionsMenu } from '@/shell/directory_row_actions_menu';
import {
  DirectorySelectOverviewTable,
  directoryOperateRows,
  directoryRecordMap,
} from '@/shell/directory_select_overview_table';
import { TableHost } from '@/shell/ui_bands';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsListFooter } from '@/domains/ops/ops_list_footer';
import { OpsPageBlockingError, OpsPageLoading, OpsPageShell } from '@/domains/ops/ops_page_shell';
import { OpsStatusChip } from '@/domains/ops/ops_status';

export type OpsDlqInboxProps = {
  items?: DLQInboxEntry[];
  nextCursor?: string;
  partial?: boolean;
  limit: number;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  retryingId?: string;
  retryError?: Error;
  onPrev: () => void;
  onNext: () => void;
  canGoPrev: boolean;
  onRetry: (entry: DLQInboxEntry) => void;
  embedded?: boolean;
};

function dlqInboxRowId(entry: DLQInboxEntry): string | undefined {
  if (entry.id) {
    return entry.id;
  }
  if (entry.source && entry.failed_at) {
    return `${entry.source}-${entry.failed_at}`;
  }
  return entry.source ?? entry.failed_at ?? undefined;
}

function dlqInboxRowLabel(entry: DLQInboxEntry): string {
  if (entry.source && entry.event_type) {
    return `${entry.source} / ${entry.event_type}`;
  }
  return entry.id ?? entry.source ?? entry.event_type ?? 'DLQ entry';
}

function buildDlqInboxOverviewFields(entry: DLQInboxEntry): DirectoryOverviewField[] {
  return [
    {
      label: 'ID',
      value: (
        <span className={cn(adminTypography.monoData, 'text-muted-foreground')}>
          {entry.id ?? '-'}
        </span>
      ),
    },
    { label: 'Source', value: entry.source ?? '-' },
    {
      label: 'Status',
      value: entry.status ? <OpsStatusChip status={entry.status} /> : '-',
    },
    {
      label: 'Campaign ID',
      value: (
        <span className={cn(adminTypography.monoData, 'text-muted-foreground')}>
          {entry.campaign_id ?? '-'}
        </span>
      ),
    },
    { label: 'Event type', value: entry.event_type ?? '-' },
    { label: 'Error', value: entry.error ?? '-' },
    {
      label: 'Failed',
      value: displayTimestamp(entry.failed_at, entry.failed_at_display) || '-',
    },
    { label: 'Retries', value: entry.retry_count ?? '-' },
    { label: 'Shard ID', value: entry.shard_id ?? '-' },
  ];
}

function DlqInboxTable({
  items,
  fetching,
  retryingId,
  onRetry,
}: {
  items: DLQInboxEntry[];
  fetching: boolean;
  retryingId?: string;
  onRetry: (entry: DLQInboxEntry) => void;
}) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const recordById = useMemo(() => directoryRecordMap(items, dlqInboxRowId), [items]);
  const rows = useMemo(() => directoryOperateRows(items, dlqInboxRowId, dlqInboxRowLabel), [items]);

  return (
    <TableHost>
      <DirectorySelectOverviewTable
        buildOverviewFields={buildDlqInboxOverviewFields}
        disabled={fetching}
        nameColumnLabel="DLQ entry"
        overviewTitle={(entry) => dlqInboxRowLabel(entry)}
        recordById={recordById}
        renderActions={(row, entry, openOverview) => {
          const canRetry = Boolean(entry.id && entry.source);
          if (!canRetry) {
            return (
              <DirectoryRowActionsMenu
                ariaLabel={`DLQ entry ${row.label}`}
                disabled={fetching}
                onOverview={openOverview}
              />
            );
          }
          return (
            <DirectoryRowActionsMenu
              ariaLabel={`DLQ entry ${row.label}`}
              disabled={fetching || retryingId === entry.id}
              onOverview={openOverview}
            >
              <DropdownMenuItem
                disabled={fetching || retryingId === entry.id}
                onClick={() => onRetry(entry)}
              >
                {retryingId === entry.id ? 'Retrying...' : 'Retry'}
              </DropdownMenuItem>
            </DirectoryRowActionsMenu>
          );
        }}
        rows={rows}
        selectedId={selectedId}
        onSelectedIdChange={setSelectedId}
      />
    </TableHost>
  );
}

export function OpsDlqInbox({
  items,
  nextCursor,
  partial,
  fetching,
  error,
  hasSnapshot,
  retryingId,
  retryError,
  onPrev,
  onNext,
  canGoPrev,
  onRetry,
  embedded = false,
}: OpsDlqInboxProps) {
  const list = items ?? [];
  const footer = (
    <OpsListFooter
      canGoNext={Boolean(nextCursor)}
      canGoPrev={canGoPrev}
      disabled={fetching}
      summary={`${list.length} entries on this page${nextCursor ? '  /  more pages available' : ''}`}
      onNext={onNext}
      onPrev={onPrev}
    />
  );

  const body =
    list.length === 0 ? (
      <EmptyState description="No failed deliveries are queued." title="DLQ inbox empty" />
    ) : (
      <DlqInboxTable fetching={fetching} items={list} retryingId={retryingId} onRetry={onRetry} />
    );

  if (embedded) {
    return (
      <>
        {body}
        {footer}
        {retryError ? opsPanelError(retryError, 'Retry failed') : null}
      </>
    );
  }

  if (fetching && !hasSnapshot && !error) {
    return <OpsPageLoading />;
  }

  if (error && !hasSnapshot) {
    return (
      <OpsPageBlockingError error={error} pageTitle="DLQ inbox" title="Could not load DLQ inbox" />
    );
  }

  return (
    <OpsPageShell
      badge={partial ? <OpsStatusChip status="partial" /> : undefined}
      footer={footer}
      title="DLQ inbox"
    >
      {body}
      {retryError ? opsPanelError(retryError, 'Retry failed') : null}
      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </OpsPageShell>
  );
}
