import { PrimaryActionButton, SecondaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { EmptyState } from '@/components/system/empty_state';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { PaginationPrevNext } from '@/components/system/pagination_prev_next';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { OpsBlacklistEntry } from '@/api/types';
import { OpsNav } from '@/domains/ops/ops_nav';
import { displayTimestamp } from '@/lib/display';

export type OpsBlacklistProps = {
  items: OpsBlacklistEntry[];
  total: number;
  limit: number;
  offset: number;
  draftIp: string;
  draftReason: string;
  draftRemoveIp: string;
  fetching: boolean;
  saving: boolean;
  error: Error | undefined;
  actionError: Error | undefined;
  hasSnapshot: boolean;
  onDraftIpChange: (value: string) => void;
  onDraftReasonChange: (value: string) => void;
  onDraftRemoveIpChange: (value: string) => void;
  onAdd: () => void;
  onRemove: () => void;
  onPageChange: (nextOffset: number) => void;
};

export function OpsBlacklist({
  items,
  total,
  limit,
  offset,
  draftIp,
  draftReason,
  draftRemoveIp,
  fetching,
  saving,
  error,
  actionError,
  hasSnapshot,
  onDraftIpChange,
  onDraftReasonChange,
  onDraftRemoveIpChange,
  onAdd,
  onRemove,
  onPageChange,
}: OpsBlacklistProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Fraud blacklist">
        <OpsNav />
        <ErrorBlock title="Could not load blacklist" message={error.message} />
      </PageChrome>
    );
  }

  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;

  return (
    <PageChrome title="Fraud blacklist">
      <OpsNav />

      <div className="ui-filter-panel md:grid-cols-[repeat(auto-fill,minmax(12rem,1fr))]">
        <div className="grid gap-2">
          <Label htmlFor="blacklist-ip">IP to block</Label>
          <Input id="blacklist-ip" value={draftIp} onChange={(event) => onDraftIpChange(event.target.value)} />
        </div>
        <div className="grid gap-2 md:col-span-2">
          <Label htmlFor="blacklist-reason">Reason</Label>
          <Input
            id="blacklist-reason"
            value={draftReason}
            onChange={(event) => onDraftReasonChange(event.target.value)}
          />
        </div>
        <PrimaryActionButton disabled={!draftIp.trim()} loading={saving} onClick={onAdd} type="button">
          Block IP
        </PrimaryActionButton>
      </div>

      <div className="grid max-w-md grid-cols-[1fr_auto] items-end gap-4">
        <div className="grid gap-2">
          <Label htmlFor="blacklist-remove-ip">IP to unblock</Label>
          <Input
            id="blacklist-remove-ip"
            value={draftRemoveIp}
            onChange={(event) => onDraftRemoveIpChange(event.target.value)}
          />
        </div>
        <SecondaryActionButton disabled={!draftRemoveIp.trim()} loading={saving} onClick={onRemove} type="button">
          Unblock
        </SecondaryActionButton>
      </div>

      <form
        className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4"
        onSubmit={(event) => event.preventDefault()}
      >
        <div className="col-span-full text-sm text-muted-foreground">
          {total > 0
            ? `Showing ${offset + 1}-${Math.min(offset + items.length, total)} of ${total}`
            : 'No blacklist entries'}
        </div>
        <PaginationPrevNext
          canGoPrev={canGoPrev}
          canGoNext={canGoNext}
          disabled={fetching}
          onPrev={() => onPageChange(Math.max(0, offset - limit))}
          onNext={() => onPageChange(offset + limit)}
        />
      </form>

      {items.length === 0 ? (
        <EmptyState title="Blacklist empty" description="No blocked IPs on record." />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>IP</TableHead>
                <TableHead>Reason</TableHead>
                <TableHead>Created</TableHead>
                <TableHead>Expires</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((row) => (
                <TableRow key={row.id ?? row.ip}>
                  <TableCell>{row.ip ?? ''}</TableCell>
                  <TableCell>{row.reason ?? ''}</TableCell>
                  <TableCell>{displayTimestamp(row.created_at, row.created_at_display)}</TableCell>
                  <TableCell>{displayTimestamp(row.expires_at, row.expires_at_display)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {actionError ? <ErrorBlock title="Action failed" message={actionError.message} /> : null}
      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageChrome>
  );
}
