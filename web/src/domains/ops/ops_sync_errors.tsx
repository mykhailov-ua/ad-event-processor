import { useState } from 'react';

import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import type { PostbackDlqEntry } from '@/api/types';
import { EmptyState } from '@/shell/empty_state';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
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
  return (
    <>
      {retryError ? opsPanelError(retryError, 'Postback retry failed') : null}
      {dlq.length === 0 ? (
        <EmptyState
          description="No failed postback deliveries are queued."
          title="Postbacks DLQ empty"
        />
      ) : (
        <DirectoryTable horizontalScroll>
          <TableHeader>
            <TableRow>
              <DirectoryTableHead>ID</DirectoryTableHead>
              <DirectoryTableHead>Campaign</DirectoryTableHead>
              <DirectoryTableHead>Click ID</DirectoryTableHead>
              <DirectoryTableHead>Event</DirectoryTableHead>
              <DirectoryTableHead>Status</DirectoryTableHead>
              <DirectoryTableHead>Failures</DirectoryTableHead>
              <DirectoryTableHead>Last error</DirectoryTableHead>
              <DirectoryTableHead>Actions</DirectoryTableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {dlq.map((row) => {
              const rowId = row.id != null ? String(row.id) : '';
              return (
                <TableRow key={rowId || row.campaign_id}>
                  <TableCell>{row.id}</TableCell>
                  <TableCell>{row.campaign_id ?? ''}</TableCell>
                  <TableCell>{row.click_id ?? ''}</TableCell>
                  <TableCell>{row.event_type ?? ''}</TableCell>
                  <TableCell>{row.status ?? ''}</TableCell>
                  <TableCell>{row.failures_count ?? ''}</TableCell>
                  <TableCell>{row.last_error ?? ''}</TableCell>
                  <TableCell>
                    <Button
                      disabled={!rowId || fetching || retryingId === rowId}
                      onClick={() => {
                        if (rowId) {
                          onRetry(rowId);
                        }
                      }}
                      type="button"
                      variant="outline"
                    >
                      {retryingId === rowId ? 'Retrying...' : 'Retry'}
                    </Button>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </DirectoryTable>
      )}
    </>
  );
}
