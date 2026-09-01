import { useEffect, useState } from 'react';

import { PrimaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { RowActionsMenu } from '@/components/system/row_actions_menu';
import { CustomerScopeBar } from '@/components/system/customer_scope_bar';
import { EmptyState } from '@/components/system/empty_state';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
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
import type { SmartAlertRule } from '@/api/types';
import {
  type SmartAlertRuleEditDraft,
  smartAlertRuleEditFromRow,
} from '@/domains/automation/automation_rule_forms';
import { AutomationNav, automationPanelError } from '@/domains/automation/automation_nav';

export type SmartAlertsRulesDirectoryProps = {
  items: SmartAlertRule[];
  appliedCustomerId: string;
  draftCustomerId: string;
  createDraft: SmartAlertRuleEditDraft;
  ruleDrafts: Record<string, SmartAlertRuleEditDraft>;
  fetching: boolean;
  creating: boolean;
  error: Error | undefined;
  actionError: Error | undefined;
  createSuccess: boolean;
  hasSnapshot: boolean;
  updatingRuleId: string | undefined;
  deletingRuleId: string | undefined;
  onDraftCustomerIdChange: (value: string) => void;
  onApplyCustomerScope: () => void;
  onCreateDraftChange: (patch: Partial<SmartAlertRuleEditDraft>) => void;
  onRuleDraftChange: (ruleId: string, patch: Partial<SmartAlertRuleEditDraft>) => void;
  onCreate: () => void;
  onSaveRule: (ruleId: string) => void;
  onDeleteRule: (ruleId: string) => void;
};

export function SmartAlertsRulesDirectory({
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
  updatingRuleId,
  deletingRuleId,
  onDraftCustomerIdChange,
  onApplyCustomerScope,
  onCreateDraftChange,
  onRuleDraftChange,
  onCreate,
  onSaveRule,
  onDeleteRule,
}: SmartAlertsRulesDirectoryProps) {
  const [createOpen, setCreateOpen] = useState(false);

  useEffect(() => {
    if (createSuccess) {
      setCreateOpen(false);
    }
  }, [createSuccess]);

  if (!appliedCustomerId) {
    return (
      <PageChrome title="Smart alert rules">
        <AutomationNav />
        <CustomerScopeBar
          appliedCustomerId={appliedCustomerId}
          draftCustomerId={draftCustomerId}
          onApply={onApplyCustomerScope}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
        />
        <EmptyState
          title="Customer required"
          description="Apply a customer ID to list smart alert rules."
        />
      </PageChrome>
    );
  }

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Smart alert rules">
        <AutomationNav />
        <CustomerScopeBar
          appliedCustomerId={appliedCustomerId}
          draftCustomerId={draftCustomerId}
          onApply={onApplyCustomerScope}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
        />
        {automationPanelError(error, 'Could not load smart alert rules')}
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="Smart alert rules"
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
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Create rule</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 md:grid-cols-[repeat(auto-fill,minmax(12rem,1fr))]">
            <div className="grid gap-2">
              <Label htmlFor="smart-alert-create-name">Name</Label>
              <Input
                id="smart-alert-create-name"
                value={createDraft.name}
                onChange={(event) => onCreateDraftChange({ name: event.target.value })}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="smart-alert-create-metric">Metric</Label>
              <Input
                id="smart-alert-create-metric"
                value={createDraft.metric}
                onChange={(event) => onCreateDraftChange({ metric: event.target.value })}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="smart-alert-create-operator">Operator</Label>
              <Input
                id="smart-alert-create-operator"
                value={createDraft.operator}
                onChange={(event) => onCreateDraftChange({ operator: event.target.value })}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="smart-alert-create-threshold">Threshold</Label>
              <Input
                id="smart-alert-create-threshold"
                inputMode="decimal"
                value={createDraft.threshold}
                onChange={(event) => onCreateDraftChange({ threshold: event.target.value })}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="smart-alert-create-window">Window (min)</Label>
              <Input
                id="smart-alert-create-window"
                inputMode="numeric"
                value={createDraft.window_minutes}
                onChange={(event) => onCreateDraftChange({ window_minutes: event.target.value })}
              />
            </div>
            <div className="grid gap-2 md:col-span-2">
              <Label htmlFor="smart-alert-create-webhook">Webhook URL</Label>
              <Input
                id="smart-alert-create-webhook"
                value={createDraft.webhook_url}
                onChange={(event) => onCreateDraftChange({ webhook_url: event.target.value })}
              />
            </div>
            <div className="flex items-center gap-2 md:col-span-2">
              <Checkbox
                checked={createDraft.enabled}
                id="smart-alert-create-enabled"
                onCheckedChange={(checked) => onCreateDraftChange({ enabled: checked === true })}
              />
              <Label htmlFor="smart-alert-create-enabled">Enabled</Label>
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
        <EmptyState title="No alert rules" description="No smart alert rules for this customer." />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Metric</TableHead>
                <TableHead>Operator</TableHead>
                <TableHead>Threshold</TableHead>
                <TableHead>Window</TableHead>
                <TableHead>Webhook</TableHead>
                <TableHead>Enabled</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((row) => {
                const ruleId = row.id ?? '';
                const draft = ruleDrafts[ruleId] ?? smartAlertRuleEditFromRow(row);
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
                    <TableCell>
                      <Input
                        aria-label={`Metric for rule ${ruleId}`}
                        className="min-w-[6rem]"
                        value={draft.metric}
                        onChange={(event) =>
                          onRuleDraftChange(ruleId, { metric: event.target.value })
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        aria-label={`Operator for rule ${ruleId}`}
                        className="min-w-[4rem]"
                        value={draft.operator}
                        onChange={(event) =>
                          onRuleDraftChange(ruleId, { operator: event.target.value })
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        aria-label={`Threshold for rule ${ruleId}`}
                        className="min-w-[5rem]"
                        inputMode="decimal"
                        value={draft.threshold}
                        onChange={(event) =>
                          onRuleDraftChange(ruleId, { threshold: event.target.value })
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        aria-label={`Window for rule ${ruleId}`}
                        className="min-w-[4rem]"
                        inputMode="numeric"
                        value={draft.window_minutes}
                        onChange={(event) =>
                          onRuleDraftChange(ruleId, { window_minutes: event.target.value })
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Input
                        aria-label={`Webhook for rule ${ruleId}`}
                        className="min-w-[10rem] font-mono text-xs"
                        value={draft.webhook_url}
                        onChange={(event) =>
                          onRuleDraftChange(ruleId, { webhook_url: event.target.value })
                        }
                      />
                    </TableCell>
                    <TableCell>
                      <Checkbox
                        aria-label={`Enabled for rule ${ruleId}`}
                        checked={draft.enabled}
                        onCheckedChange={(checked) =>
                          onRuleDraftChange(ruleId, { enabled: checked === true })
                        }
                      />
                    </TableCell>
                    <TableCell className="text-right">
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

      {error && hasSnapshot ? automationPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
