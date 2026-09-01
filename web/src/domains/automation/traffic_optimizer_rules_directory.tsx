import { useEffect, useState } from 'react';

import { PrimaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { RowActionsMenu } from '@/components/system/row_actions_menu';
import { CustomerScopeBar } from '@/components/system/customer_scope_bar';
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
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
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
import type { TrafficOptimizerDryRunResult, TrafficOptimizerRule } from '@/api/types';
import {
  type TrafficOptimizerRuleEditDraft,
  trafficOptimizerRuleEditFromRow,
} from '@/domains/automation/automation_rule_forms';
import { AutomationNav, automationPanelError } from '@/domains/automation/automation_nav';

export type TrafficOptimizerRulesDirectoryProps = {
  items: TrafficOptimizerRule[];
  appliedCustomerId: string;
  draftCustomerId: string;
  createDraft: TrafficOptimizerRuleEditDraft;
  ruleDrafts: Record<string, TrafficOptimizerRuleEditDraft>;
  fetching: boolean;
  creating: boolean;
  error: Error | undefined;
  actionError: Error | undefined;
  createSuccess: boolean;
  hasSnapshot: boolean;
  dryRunRuleId: string | undefined;
  dryRunResult: TrafficOptimizerDryRunResult | undefined;
  dryRunError: Error | undefined;
  dryRunning: boolean;
  updatingRuleId: string | undefined;
  deletingRuleId: string | undefined;
  onDraftCustomerIdChange: (value: string) => void;
  onApplyCustomerScope: () => void;
  onCreateDraftChange: (patch: Partial<TrafficOptimizerRuleEditDraft>) => void;
  onRuleDraftChange: (ruleId: string, patch: Partial<TrafficOptimizerRuleEditDraft>) => void;
  onCreate: () => void;
  onSaveRule: (ruleId: string) => void;
  onDeleteRule: (ruleId: string) => void;
  onDryRun: (ruleId: string) => void;
};

export function TrafficOptimizerRulesDirectory({
  items,
  appliedCustomerId,
  draftCustomerId,
  createDraft,
  ruleDrafts,
  fetching,
  creating,
  error,
  actionError,
  createSuccess,
  hasSnapshot,
  dryRunRuleId,
  dryRunResult,
  dryRunError,
  dryRunning,
  updatingRuleId,
  deletingRuleId,
  onDraftCustomerIdChange,
  onApplyCustomerScope,
  onCreateDraftChange,
  onRuleDraftChange,
  onCreate,
  onSaveRule,
  onDeleteRule,
  onDryRun,
}: TrafficOptimizerRulesDirectoryProps) {
  const [createOpen, setCreateOpen] = useState(false);

  useEffect(() => {
    if (createSuccess) {
      setCreateOpen(false);
    }
  }, [createSuccess]);

  if (!appliedCustomerId) {
    return (
      <PageChrome title="Traffic optimizer rules">
        <AutomationNav />
        <CustomerScopeBar
          appliedCustomerId={appliedCustomerId}
          draftCustomerId={draftCustomerId}
          onApply={onApplyCustomerScope}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
        />
        <EmptyState
          title="Customer required"
          description="Apply a customer ID to list traffic optimizer rules."
        />
      </PageChrome>
    );
  }

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Traffic optimizer rules">
        <AutomationNav />
        <CustomerScopeBar
          appliedCustomerId={appliedCustomerId}
          draftCustomerId={draftCustomerId}
          onApply={onApplyCustomerScope}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
        />
        {automationPanelError(error, 'Could not load optimizer rules')}
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="Traffic optimizer rules"
      actions={
        <PrimaryActionButton onClick={() => setCreateOpen(true)} type="button">
          Create rule
        </PrimaryActionButton>
      }
    >
      <AutomationNav />

      <CustomerScopeBar
        appliedCustomerId={appliedCustomerId}
        draftCustomerId={draftCustomerId}
        onApply={onApplyCustomerScope}
        onDraftCustomerIdChange={onDraftCustomerIdChange}
      />

      <Dialog onOpenChange={setCreateOpen} open={createOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Create rule</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="optimizer-create-name">Name</Label>
              <Input
                id="optimizer-create-name"
                value={createDraft.name}
                onChange={(event) => onCreateDraftChange({ name: event.target.value })}
              />
            </div>
            <div className="flex items-center gap-2">
              <Checkbox
                checked={createDraft.enabled}
                id="optimizer-create-enabled"
                onCheckedChange={(checked) => onCreateDraftChange({ enabled: checked === true })}
              />
              <Label htmlFor="optimizer-create-enabled">Enabled</Label>
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
        <EmptyState title="No optimizer rules" description="No rules exist for this customer." />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Scope</TableHead>
                <TableHead>Objective</TableHead>
                <TableHead>Algorithm</TableHead>
                <TableHead>Enabled</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((row) => {
                const ruleId = row.id ?? '';
                const draft = ruleDrafts[ruleId] ?? trafficOptimizerRuleEditFromRow(row);
                const updating = updatingRuleId === ruleId;
                const deleting = deletingRuleId === ruleId;
                return (
                  <TableRow key={ruleId}>
                    <TableCell>
                      <Input
                        aria-label={`Name for rule ${ruleId}`}
                        className="min-w-[8rem]"
                        value={draft.name}
                        onChange={(event) =>
                          onRuleDraftChange(ruleId, { name: event.target.value })
                        }
                      />
                    </TableCell>
                    <TableCell>{row.scope}</TableCell>
                    <TableCell>{row.objective}</TableCell>
                    <TableCell>{row.algorithm}</TableCell>
                    <TableCell>
                      <Checkbox
                        aria-label={`Enabled for rule ${ruleId}`}
                        checked={draft.enabled}
                        onCheckedChange={(checked) =>
                          onRuleDraftChange(ruleId, { enabled: checked === true })
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <RowActionsMenu
                        ariaLabel="Rule actions"
                        disabled={!ruleId || updating || deleting}
                      >
                        <DropdownMenuItem
                          disabled={!ruleId || updating || deleting}
                          onClick={() => onSaveRule(ruleId)}
                        >
                          {updating ? 'Saving...' : 'Save'}
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          disabled={(dryRunning && dryRunRuleId === ruleId) || !ruleId || deleting}
                          onClick={() => onDryRun(ruleId)}
                        >
                          Dry-run
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          className="text-destructive focus:text-destructive"
                          disabled={!ruleId || updating || deleting}
                          onClick={() => onDeleteRule(ruleId)}
                        >
                          {deleting ? 'Deleting...' : 'Delete'}
                        </DropdownMenuItem>
                      </RowActionsMenu>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}

      {actionError ? <ErrorBlock title="Action failed" message={actionError.message} /> : null}

      {dryRunError ? automationPanelError(dryRunError, 'Dry-run failed') : null}

      {dryRunResult ? (
        <section className="grid gap-2">
          <h2 className="text-base font-semibold">Dry-run result</h2>
          {dryRunResult.stale_weights ? (
            <Badge variant="secondary">Stale weights</Badge>
          ) : null}
          {(dryRunResult.arms ?? []).length > 0 ? (
            <div className="ui-table-frame">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Entity</TableHead>
                    <TableHead>Current weight</TableHead>
                    <TableHead>Proposed weight</TableHead>
                    <TableHead>Observed value</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(dryRunResult.arms ?? []).map((arm) => (
                    <TableRow key={arm.entity_id}>
                      <TableCell className="font-mono text-xs">{arm.entity_id}</TableCell>
                      <TableCell>{arm.current_weight}</TableCell>
                      <TableCell>{arm.proposed_weight}</TableCell>
                      <TableCell>{arm.observed_value}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : null}
        </section>
      ) : null}

      {error && hasSnapshot ? automationPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
