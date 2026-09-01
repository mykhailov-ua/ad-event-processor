import { useEffect, useState } from 'react';

import { PrimaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { CampaignScopeBar } from '@/components/system/campaign_scope_bar';
import { EmptyState } from '@/components/system/empty_state';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from '@/components/ui/checkbox';
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
import type { MarginGuardPolicy } from '@/api/types';
import { AutomationNav, automationPanelError } from '@/domains/automation/automation_nav';

export type PolicyCreateDraft = {
  name: string;
  roi_floor_pct: string;
  min_clicks: string;
  zero_conv_streak: string;
  cost_over_revenue_threshold_bps: string;
  is_active: boolean;
};

export type MarginGuardPoliciesDirectoryProps = {
  items: MarginGuardPolicy[];
  appliedCampaignId: string;
  draftCampaignId: string;
  createDraft: PolicyCreateDraft;
  fetching: boolean;
  creating: boolean;
  error: Error | undefined;
  actionError: Error | undefined;
  createSuccess: boolean;
  hasSnapshot: boolean;
  onDraftCampaignIdChange: (value: string) => void;
  onApplyCampaignScope: () => void;
  onCreateDraftChange: (patch: Partial<PolicyCreateDraft>) => void;
  onCreate: () => void;
};

export function MarginGuardPoliciesDirectory({
  items,
  appliedCampaignId,
  draftCampaignId,
  createDraft,
  fetching,
  creating,
  error,
  actionError,
  createSuccess,
  hasSnapshot,
  onDraftCampaignIdChange,
  onApplyCampaignScope,
  onCreateDraftChange,
  onCreate,
}: MarginGuardPoliciesDirectoryProps) {
  const [createOpen, setCreateOpen] = useState(false);

  useEffect(() => {
    if (createSuccess) {
      setCreateOpen(false);
    }
  }, [createSuccess]);

  if (!appliedCampaignId) {
    return (
      <PageChrome title="Margin guard policies">
        <AutomationNav />
        <CampaignScopeBar
          appliedCampaignId={appliedCampaignId}
          draftCampaignId={draftCampaignId}
          onApply={onApplyCampaignScope}
          onDraftCampaignIdChange={onDraftCampaignIdChange}
        />
        <EmptyState
          title="Campaign required"
          description="Apply a campaign ID to list margin guard policies."
        />
      </PageChrome>
    );
  }

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Margin guard policies">
        <AutomationNav />
        <CampaignScopeBar
          appliedCampaignId={appliedCampaignId}
          draftCampaignId={draftCampaignId}
          onApply={onApplyCampaignScope}
          onDraftCampaignIdChange={onDraftCampaignIdChange}
        />
        {automationPanelError(error, 'Could not load margin guard policies')}
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="Margin guard policies"
      actions={
        <PrimaryActionButton onClick={() => setCreateOpen(true)} type="button">
          Create policy
        </PrimaryActionButton>
      }
    >
      <AutomationNav />

      <CampaignScopeBar
        appliedCampaignId={appliedCampaignId}
        draftCampaignId={draftCampaignId}
        onApply={onApplyCampaignScope}
        onDraftCampaignIdChange={onDraftCampaignIdChange}
      />

      <Dialog onOpenChange={setCreateOpen} open={createOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Create policy</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 md:grid-cols-[repeat(auto-fill,minmax(12rem,1fr))]">
            <div className="grid gap-2">
              <Label htmlFor="margin-guard-create-name">Name</Label>
              <Input
                id="margin-guard-create-name"
                value={createDraft.name}
                onChange={(event) => onCreateDraftChange({ name: event.target.value })}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="margin-guard-create-roi">ROI floor %</Label>
              <Input
                id="margin-guard-create-roi"
                inputMode="decimal"
                value={createDraft.roi_floor_pct}
                onChange={(event) => onCreateDraftChange({ roi_floor_pct: event.target.value })}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="margin-guard-create-min-clicks">Min clicks</Label>
              <Input
                id="margin-guard-create-min-clicks"
                inputMode="numeric"
                value={createDraft.min_clicks}
                onChange={(event) => onCreateDraftChange({ min_clicks: event.target.value })}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="margin-guard-create-zero-streak">Zero conv streak</Label>
              <Input
                id="margin-guard-create-zero-streak"
                inputMode="numeric"
                value={createDraft.zero_conv_streak}
                onChange={(event) => onCreateDraftChange({ zero_conv_streak: event.target.value })}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="margin-guard-create-cost-bps">Cost/revenue bps</Label>
              <Input
                id="margin-guard-create-cost-bps"
                inputMode="numeric"
                value={createDraft.cost_over_revenue_threshold_bps}
                onChange={(event) =>
                  onCreateDraftChange({ cost_over_revenue_threshold_bps: event.target.value })
                }
              />
            </div>
            <div className="flex items-center gap-2 md:col-span-2">
              <Checkbox
                checked={createDraft.is_active}
                id="margin-guard-create-active"
                onCheckedChange={(checked) => onCreateDraftChange({ is_active: checked === true })}
              />
              <Label htmlFor="margin-guard-create-active">Active</Label>
            </div>
          </div>
          <DialogFooter>
            <PrimaryActionButton
              disabled={!createDraft.name.trim()}
              loading={creating}
              onClick={onCreate}
              type="button"
            >
              Create
            </PrimaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {items.length === 0 ? (
        <EmptyState title="No policies" description="No margin guard policies for this campaign." />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>ROI floor %</TableHead>
                <TableHead>Min clicks</TableHead>
                <TableHead>Active</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((row) => (
                <TableRow key={row.id ?? row.name}>
                  <TableCell>{row.name}</TableCell>
                  <TableCell>{row.roi_floor_pct}</TableCell>
                  <TableCell>{row.min_clicks}</TableCell>
                  <TableCell>
                    <Badge variant={row.is_active ? 'default' : 'outline'}>
                      {row.is_active ? 'yes' : 'no'}
                    </Badge>
                  </TableCell>
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
