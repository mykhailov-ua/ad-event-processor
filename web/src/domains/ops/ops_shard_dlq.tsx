import { useMemo, useState } from 'react';

import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import type { DLQEntry } from '@/api/types';
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

export type OpsShardDlqProps = {
  items?: DLQEntry[];
  nextCursor?: string;
  fetching: boolean;
  retryingId?: string;
  retryError?: Error;
  onPrev: () => void;
  onNext: () => void;
  canGoPrev: boolean;
  onRetry: (entry: DLQEntry) => void;
};

function shardDlqRowId(entry: DLQEntry): string | undefined {
  return entry.id ?? undefined;
}

function shardDlqRowLabel(entry: DLQEntry): string {
  if (entry.shard_id != null && entry.event_type) {
    return `Shard ${entry.shard_id} / ${entry.event_type}`;
  }
  return entry.id ?? `Shard ${entry.shard_id ?? '?'}`;
}

function buildShardDlqOverviewFields(entry: DLQEntry): DirectoryOverviewField[] {
  return [
    {
      label: 'ID',
      value: (
        <span className={cn(adminTypography.monoData, 'text-muted-foreground')}>
          {entry.id ?? '-'}
        </span>
      ),
    },
    { label: 'Shard', value: entry.shard_id ?? '-' },
    { label: 'Stream', value: entry.stream_id ?? '-' },
    {
      label: 'Entry ID',
      value: (
        <span className={cn(adminTypography.monoData, 'text-muted-foreground')}>
          {entry.entry_id ?? '-'}
        </span>
      ),
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
      value: displayTimestamp(entry.failed_at, undefined) || '-',
    },
    { label: 'Retries', value: entry.retry_count ?? '-' },
  ];
}

export function OpsShardDlq({
  items,
  nextCursor,
  fetching,
  retryingId,
  retryError,
  onPrev,
  onNext,
  canGoPrev,
  onRetry,
}: OpsShardDlqProps) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const list = items ?? [];
  const recordById = useMemo(() => directoryRecordMap(list, shardDlqRowId), [list]);
  const rows = useMemo(() => directoryOperateRows(list, shardDlqRowId, shardDlqRowLabel), [list]);

  return (
    <>
      {retryError ? opsPanelError(retryError, 'Retry failed') : null}
      {list.length === 0 ? (
        <EmptyState
          description="No shard fan-out DLQ entries on this page."
          title="Shard DLQ empty"
        />
      ) : (
        <TableHost>
          <DirectorySelectOverviewTable
            buildOverviewFields={buildShardDlqOverviewFields}
            disabled={fetching}
            nameColumnLabel="Shard DLQ entry"
            overviewTitle={(entry) => shardDlqRowLabel(entry)}
            recordById={recordById}
            renderActions={(row, entry, openOverview) => {
              const canRetry = Boolean(entry.id);
              if (!canRetry) {
                return (
                  <DirectoryRowActionsMenu
                    ariaLabel={`Shard DLQ entry ${row.label}`}
                    disabled={fetching}
                    onOverview={openOverview}
                  />
                );
              }
              return (
                <DirectoryRowActionsMenu
                  ariaLabel={`Shard DLQ entry ${row.label}`}
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
      )}
      <OpsListFooter
        canGoNext={Boolean(nextCursor)}
        canGoPrev={canGoPrev}
        disabled={fetching}
        summary={`${list.length} entries on this page${nextCursor ? '  /  more pages available' : ''}`}
        onNext={onNext}
        onPrev={onPrev}
      />
    </>
  );
}
