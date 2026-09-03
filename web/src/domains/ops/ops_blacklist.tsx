import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { EmptyState } from '@/shell/empty_state';
import type { OpsBlacklistEntry } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsListFooter } from '@/domains/ops/ops_list_footer';
import {
  OpsActionGroup,
  OpsPageBlockingError,
  OpsPageLoading,
  OpsPageShell,
} from '@/domains/ops/ops_page_shell';
import {
  OpsTable,
  OpsTableCell,
  OpsTableHead,
  OpsTableHeaderRow,
  OpsTableRow,
} from '@/domains/ops/ops_table';

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
    return <OpsPageLoading />;
  }

  if (error && !hasSnapshot) {
    return (
      <OpsPageBlockingError
        error={error}
        pageTitle="Fraud blacklist"
        title="Could not load blacklist"
      />
    );
  }

  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;

  return (
    <OpsPageShell
      actions={
        <>
          <OpsActionGroup label="Block">
            <Button disabled={!draftIp.trim() || saving} loading={saving} type="button" onClick={onAdd}>
              Block IP
            </Button>
          </OpsActionGroup>
          <OpsActionGroup label="Unblock">
            <Button
              disabled={!draftRemoveIp.trim() || saving}
              loading={saving}
              type="button"
              onClick={onRemove}
            >
              Unblock
            </Button>
          </OpsActionGroup>
        </>
      }
      filters={
        <>
          <div className="grid gap-2">
            <Label htmlFor="blacklist-ip">IP to block</Label>
            <Input
              id="blacklist-ip"
              value={draftIp}
              onChange={(event) => onDraftIpChange(event.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="blacklist-reason">Reason</Label>
            <Input
              id="blacklist-reason"
              value={draftReason}
              onChange={(event) => onDraftReasonChange(event.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="blacklist-remove-ip">IP to unblock</Label>
            <Input
              id="blacklist-remove-ip"
              value={draftRemoveIp}
              onChange={(event) => onDraftRemoveIpChange(event.target.value)}
            />
          </div>
        </>
      }
      footer={
        <OpsListFooter
          canGoNext={canGoNext}
          canGoPrev={canGoPrev}
          disabled={fetching}
          summary={
            total > 0
              ? `Showing ${offset + 1}-${Math.min(offset + items.length, total)} of ${total}`
              : 'No blacklist entries'
          }
          onNext={() => onPageChange(offset + limit)}
          onPrev={() => onPageChange(Math.max(0, offset - limit))}
        />
      }
      title="Fraud blacklist"
    >
      {items.length === 0 ? (
        <EmptyState description="No blocked IPs on record." title="Blacklist empty" />
      ) : (
        <OpsTable
          head={
            <OpsTableHeaderRow>
              <OpsTableHead>IP</OpsTableHead>
              <OpsTableHead>Reason</OpsTableHead>
              <OpsTableHead>Created</OpsTableHead>
              <OpsTableHead>Expires</OpsTableHead>
            </OpsTableHeaderRow>
          }
        >
          {items.map((row) => (
            <OpsTableRow key={row.id ?? row.ip}>
              <OpsTableCell>{row.ip ?? ''}</OpsTableCell>
              <OpsTableCell>{row.reason ?? ''}</OpsTableCell>
              <OpsTableCell>{displayTimestamp(row.created_at, row.created_at_display)}</OpsTableCell>
              <OpsTableCell>{displayTimestamp(row.expires_at, row.expires_at_display)}</OpsTableCell>
            </OpsTableRow>
          ))}
        </OpsTable>
      )}

      {actionError ? opsPanelError(actionError, 'Action failed') : null}
      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </OpsPageShell>
  );
}
