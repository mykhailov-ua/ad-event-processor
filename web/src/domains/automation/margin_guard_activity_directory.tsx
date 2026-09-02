import { useEffect, useState } from 'react';

import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { PageChrome } from '@/shell/page_chrome';
import { CampaignScopeBar } from '@/shell/campaign_scope_bar';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
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
import type { MarginGuardActivity } from '@/api/types';
import { AutomationNav, automationPanelError } from '@/domains/automation/automation_nav';
import { displayTimestamp } from '@/lib/display';

export type MarginGuardActivityDirectoryProps = {
  items: MarginGuardActivity[];
  appliedCampaignId: string;
  draftCampaignId: string;
  draftPlacementId: string;
  fetching: boolean;
  removing: boolean;
  removeSuccess: boolean;
  error: Error | undefined;
  actionError: Error | undefined;
  hasSnapshot: boolean;
  onDraftCampaignIdChange: (value: string) => void;
  onApplyCampaignScope: () => void;
  onDraftPlacementIdChange: (value: string) => void;
  onRemoveOverride: () => void;
};

export function MarginGuardActivityDirectory({
  items,
  appliedCampaignId,
  draftCampaignId,
  draftPlacementId,
  fetching,
  removing,
  removeSuccess,
  error,
  actionError,
  hasSnapshot,
  onDraftCampaignIdChange,
  onApplyCampaignScope,
  onDraftPlacementIdChange,
  onRemoveOverride,
}: MarginGuardActivityDirectoryProps) {
  const [clearOpen, setClearOpen] = useState(false);

  useEffect(() => {
    if (removeSuccess) {
      setClearOpen(false);
    }
  }, [removeSuccess]);

  if (!appliedCampaignId) {
    return (
      <PageChrome title="Margin guard activity">
        <AutomationNav />
        <CampaignScopeBar
          appliedCampaignId={appliedCampaignId}
          draftCampaignId={draftCampaignId}
          onApply={onApplyCampaignScope}
          onDraftCampaignIdChange={onDraftCampaignIdChange}
        />
        <EmptyState
          title="Campaign required"
          description="Apply a campaign ID to list margin guard activity."
        />
      </PageChrome>
    );
  }

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Margin guard activity">
        <AutomationNav />
        <CampaignScopeBar
          appliedCampaignId={appliedCampaignId}
          draftCampaignId={draftCampaignId}
          onApply={onApplyCampaignScope}
          onDraftCampaignIdChange={onDraftCampaignIdChange}
        />
        {automationPanelError(error, 'Could not load margin guard activity')}
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="Margin guard activity"
      actions={
        <SecondaryActionButton onClick={() => setClearOpen(true)} type="button">
          Clear placement override
        </SecondaryActionButton>
      }
    >
      <AutomationNav />

      <CampaignScopeBar
        appliedCampaignId={appliedCampaignId}
        draftCampaignId={draftCampaignId}
        onApply={onApplyCampaignScope}
        onDraftCampaignIdChange={onDraftCampaignIdChange}
      />

      <Dialog onOpenChange={setClearOpen} open={clearOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Clear placement override</DialogTitle>
          </DialogHeader>
          <div className="grid gap-2">
            <Label htmlFor="margin-guard-placement-id">Placement ID</Label>
            <Input
              id="margin-guard-placement-id"
              value={draftPlacementId}
              onChange={(event) => onDraftPlacementIdChange(event.target.value)}
            />
          </div>
          <DialogFooter>
            <Button
              disabled={!draftPlacementId.trim()}
              loading={removing}
              onClick={onRemoveOverride}
              shape="pill"
              type="button"
              variant="destructive"
            >
              Clear override
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {items.length === 0 ? (
        <EmptyState title="No activity" description="No margin guard activity for this campaign." />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Placement</TableHead>
                <TableHead>Action</TableHead>
                <TableHead>Reason</TableHead>
                <TableHead>Created</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((row) => (
                <TableRow key={row.id}>
                  <TableCell className="font-mono text-xs">{row.placement_id}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{row.action}</Badge>
                  </TableCell>
                  <TableCell className="max-w-md truncate">{row.reason}</TableCell>
                  <TableCell>{displayTimestamp(row.created_at)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {actionError ? <ErrorBlock title="Action failed" message={actionError.message} /> : null}

      {error && hasSnapshot ? automationPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
