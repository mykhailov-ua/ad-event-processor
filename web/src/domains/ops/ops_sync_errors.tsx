import { useMemo, useState } from 'react';

import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import type { PostbackDlqEntry } from '@/api/types';
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
import { OpsDlqInbox } from '@/domains/ops/ops_dlq_inbox';
import { OpsShardDlq } from '@/domains/ops/ops_shard_dlq';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsPageWithLoad } from '@/domains/ops/ops_page_shell';
import { OpsStatusChip } from '@/domains/ops/ops_status';
import { useOpsDlqPageWorkspace } from '@/domains/ops/use_ops_dlq_page_workspace';
import { useOpsPostbacksDlqWorkspace } from '@/domains/ops/use_ops_postbacks_dlq_workspace';
import { useOpsShardDlqWorkspace } from '@/domains/ops/use_ops_shard_dlq_workspace';

export type OpsSyncErrorsTab = 'inbox' | 'shards' | 'postbacks';

const SYNC_ERROR_TABS: { id: OpsSyncErrorsTab; label: string }[] = [
  { id: 'inbox', label: 'Platform inbox' },
  { id: 'shards', label: 'Shard fan-out' },
  { id: 'postbacks', label: 'Postbacks' },
];

function postbackDlqRowId(row: PostbackDlqEntry): string | undefined {
  return row.id != null ? String(row.id) : undefined;
}

function postbackDlqRowLabel(row: PostbackDlqEntry): string {
  if (row.id != null) {
    return String(row.id);
  }
  if (row.event_type && row.campaign_id) {
    return `${row.event_type} / ${row.campaign_id}`;
  }
  return row.event_type ?? row.campaign_id ?? 'Postback DLQ entry';
}

function buildPostbackDlqOverviewFields(row: PostbackDlqEntry): DirectoryOverviewField[] {
  return [
    {
      label: 'ID',
      value: (
        <span className={cn(adminTypography.monoData, 'text-muted-foreground')}>
          {row.id ?? '-'}
        </span>
      ),
    },
    {
      label: 'Campaign ID',
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
    { label: 'Event type', value: row.event_type ?? '-' },
    { label: 'Status', value: row.status ?? '-' },
    { label: 'Failures', value: row.failures_count ?? '-' },
    { label: 'Last error', value: row.last_error ?? '-' },
  ];
}

function PostbacksDlqPanel({
  dlq,
  fetching,
  retryingId,
  retryError,
  onRetry,
}: {
  dlq: PostbackDlqEntry[];
  fetching: boolean;
  retryingId?: string;
  retryError?: Error;
  onRetry: (rowId: string) => void;
}) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const recordById = useMemo(() => directoryRecordMap(dlq, postbackDlqRowId), [dlq]);
  const rows = useMemo(
    () => directoryOperateRows(dlq, postbackDlqRowId, postbackDlqRowLabel),
    [dlq]
  );

  return (
    <>
      {retryError ? opsPanelError(retryError, 'Postback retry failed') : null}
      {dlq.length === 0 ? (
        <EmptyState
          description="No failed postback deliveries are queued."
          title="Postbacks DLQ empty"
        />
      ) : (
        <TableHost>
          <DirectorySelectOverviewTable
            buildOverviewFields={buildPostbackDlqOverviewFields}
            disabled={fetching}
            nameColumnLabel="Postback DLQ entry"
            overviewTitle={(row) => postbackDlqRowLabel(row)}
            recordById={recordById}
            renderActions={(row, record, openOverview) => {
              const rowId = record.id != null ? String(record.id) : '';
              const canRetry = Boolean(rowId);
              if (!canRetry) {
                return (
                  <DirectoryRowActionsMenu
                    ariaLabel={`Postback DLQ entry ${row.label}`}
                    disabled={fetching}
                    onOverview={openOverview}
                  />
                );
              }
              return (
                <DirectoryRowActionsMenu
                  ariaLabel={`Postback DLQ entry ${row.label}`}
                  disabled={fetching || retryingId === rowId}
                  onOverview={openOverview}
                >
                  <DropdownMenuItem
                    disabled={fetching || retryingId === rowId}
                    onClick={() => onRetry(rowId)}
                  >
                    {retryingId === rowId ? 'Retrying...' : 'Retry'}
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
    </>
  );
}

export function OpsSyncErrors() {
  const [tab, setTab] = useState<OpsSyncErrorsTab>('inbox');
  const inbox = useOpsDlqPageWorkspace();
  const shards = useOpsShardDlqWorkspace();
  const postbacks = useOpsPostbacksDlqWorkspace();

  const active =
    tab === 'inbox'
      ? {
          fetching: inbox.fetching,
          error: inbox.error,
          hasSnapshot: inbox.hasSnapshot,
          revalidating: inbox.listRevalidating,
          blockingErrorTitle: 'Could not load platform DLQ inbox',
        }
      : tab === 'shards'
        ? {
            fetching: shards.fetching,
            error: shards.error,
            hasSnapshot: shards.hasSnapshot,
            revalidating: shards.listRevalidating,
            blockingErrorTitle: 'Could not load shard fan-out DLQ',
          }
        : {
            fetching: postbacks.fetching,
            error: postbacks.error,
            hasSnapshot: postbacks.hasSnapshot,
            revalidating: postbacks.listRevalidating,
            blockingErrorTitle: 'Could not load postbacks DLQ',
          };

  return (
    <OpsPageWithLoad
      badge={tab === 'inbox' && inbox.partial ? <OpsStatusChip status="partial" /> : undefined}
      blockingErrorTitle={active.blockingErrorTitle}
      fetchState={{
        fetching: active.fetching,
        error: active.error,
        hasSnapshot: active.hasSnapshot,
        revalidating: active.revalidating,
      }}
      title="Sync errors"
    >
      <Tabs onValueChange={(value) => setTab(value as OpsSyncErrorsTab)} value={tab}>
        <TabsList aria-label="Sync error sources">
          {SYNC_ERROR_TABS.map((item) => (
            <TabsTrigger key={item.id} value={item.id}>
              {item.label}
            </TabsTrigger>
          ))}
        </TabsList>
        <TabsContent value="inbox">
          <OpsDlqInbox {...inbox} embedded />
        </TabsContent>
        <TabsContent value="shards">
          <OpsShardDlq {...shards} />
        </TabsContent>
        <TabsContent value="postbacks">
          <PostbacksDlqPanel {...postbacks} />
        </TabsContent>
      </Tabs>
    </OpsPageWithLoad>
  );
}
