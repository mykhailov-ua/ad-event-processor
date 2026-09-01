import { useEffect, useState } from 'react';
import { Gauge, Hash, Sigma, Zap } from 'lucide-react';
import { PrimaryActionButton, SecondaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { CustomerScopeBar } from '@/components/system/customer_scope_bar';
import { EmptyState } from '@/components/system/empty_state';
import { ErrorBlock } from '@/components/system/error_block';
import { FormSectionLabel, InputWithIcon } from '@/components/system/form_shell';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import type { AutomationDryRunResult, AutomationRule } from '@/api/types';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import type { AutomationRuleEditDraft } from '@/domains/automation/automation_rule_forms';
import { AutomationRulesGrid } from '@/domains/automation/automation_rule_card';
import { AutomationNav, automationPanelError } from '@/domains/automation/automation_nav';

export type AutomationRulesDirectoryProps = {
  items: AutomationRule[];
  appliedCustomerId: string;
  draftCustomerId: string;
  createDraft: AutomationRuleEditDraft;
  ruleDrafts: Record<string, AutomationRuleEditDraft>;
  fetching: boolean;
  creating: boolean;
  error: Error | undefined;
  actionError: Error | undefined;
  createSuccess: boolean;
  hasSnapshot: boolean;
  dryRunRuleId: string | undefined;
  dryRunResult: AutomationDryRunResult | undefined;
  dryRunError: Error | undefined;
  dryRunning: boolean;
  updatingRuleId: string | undefined;
  deletingRuleId: string | undefined;
  onDraftCustomerIdChange: (value: string) => void;
  onApplyCustomerScope: () => void;
  onCreateDraftChange: (patch: Partial<AutomationRuleEditDraft>) => void;
  onRuleDraftChange: (ruleId: string, patch: Partial<AutomationRuleEditDraft>) => void;
  onCreate: () => void;
  onSaveRule: (ruleId: string) => void;
  onDeleteRule: (ruleId: string) => void;
  onDryRun: (ruleId: string) => void;
};

export function AutomationRulesDirectory({
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
}: AutomationRulesDirectoryProps) {
  const [createOpen, setCreateOpen] = useState(false);

  useEffect(() => {
    if (createSuccess) {
      setCreateOpen(false);
    }
  }, [createSuccess]);

  if (!appliedCustomerId) {
    return (
      <PageChrome title="Automation rules">
        <AutomationNav />
        <CustomerScopeBar
          appliedCustomerId={appliedCustomerId}
          draftCustomerId={draftCustomerId}
          onApply={onApplyCustomerScope}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
        />
        <EmptyState
          title="Customer required"
          description="Apply a customer ID to list automation rules."
        />
      </PageChrome>
    );
  }

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Automation rules">
        <AutomationNav />
        <CustomerScopeBar
          appliedCustomerId={appliedCustomerId}
          draftCustomerId={draftCustomerId}
          onApply={onApplyCustomerScope}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
        />
        {automationPanelError(error, 'Could not load automation rules')}
      </PageChrome>
    );
  }

  return (
    <PageChrome
      actions={
        <PrimaryActionButton onClick={() => setCreateOpen(true)} type="button">
          New rule
        </PrimaryActionButton>
      }
      description="Customer-scoped automation with server-side dry-run against delivery metrics."
      title="Automation rules"
    >
      <AutomationNav />

      <CustomerScopeBar
        appliedCustomerId={appliedCustomerId}
        draftCustomerId={draftCustomerId}
        onApply={onApplyCustomerScope}
        onDraftCustomerIdChange={onDraftCustomerIdChange}
      />

      <Dialog onOpenChange={setCreateOpen} open={createOpen}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>New automation rule</DialogTitle>
          </DialogHeader>
          <div className="grid gap-5">
            <div className="grid gap-3">
              <FormSectionLabel>Rule</FormSectionLabel>
              <InputWithIcon
                icon={Zap}
                id="automation-create-name"
                placeholder="Budget guard"
                value={createDraft.name}
                onChange={(event) => onCreateDraftChange({ name: event.target.value })}
              />
            </div>
            <div className="grid gap-3">
              <FormSectionLabel>Condition</FormSectionLabel>
              <div className="grid gap-3 sm:grid-cols-3">
                <div className="grid gap-2">
                  <Label htmlFor="automation-create-metric">Metric</Label>
                  <InputWithIcon
                    icon={Gauge}
                    id="automation-create-metric"
                    placeholder="clicks"
                    value={createDraft.metric}
                    onChange={(event) => onCreateDraftChange({ metric: event.target.value })}
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="automation-create-operator">Operator</Label>
                  <InputWithIcon
                    icon={Sigma}
                    id="automation-create-operator"
                    placeholder=">"
                    value={createDraft.operator}
                    onChange={(event) => onCreateDraftChange({ operator: event.target.value })}
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="automation-create-threshold">Threshold</Label>
                  <InputWithIcon
                    icon={Hash}
                    id="automation-create-threshold"
                    inputMode="decimal"
                    placeholder="1000"
                    value={createDraft.threshold}
                    onChange={(event) => onCreateDraftChange({ threshold: event.target.value })}
                  />
                </div>
              </div>
            </div>
            <div className="flex items-center gap-2 rounded-2xl bg-muted/25 px-4 py-3">
              <Checkbox
                checked={createDraft.enabled}
                id="automation-create-enabled"
                onCheckedChange={(checked) =>
                  onCreateDraftChange({ enabled: checked === true })
                }
              />
              <Label className="font-normal" htmlFor="automation-create-enabled">
                Enable rule after creation
              </Label>
            </div>
          </div>
          <DialogFooter className="gap-2 sm:gap-2">
            <SecondaryActionButton onClick={() => setCreateOpen(false)} type="button">
              Cancel
            </SecondaryActionButton>
            <PrimaryActionButton
              disabled={!createDraft.name.trim()}
              loading={creating}
              onClick={onCreate}
              type="button"
            >
              Save
            </PrimaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {items.length === 0 ? (
        <EmptyState title="No automation rules" description="No rules exist for this customer." />
      ) : (
        <AutomationRulesGrid
          deletingRuleId={deletingRuleId}
          dryRunRuleId={dryRunRuleId}
          dryRunning={dryRunning}
          items={items}
          onDeleteRule={onDeleteRule}
          onDryRun={onDryRun}
          onRuleDraftChange={onRuleDraftChange}
          onSaveRule={onSaveRule}
          ruleDrafts={ruleDrafts}
          updatingRuleId={updatingRuleId}
        />
      )}

      {actionError ? <ErrorBlock title="Action failed" message={actionError.message} /> : null}

      {dryRunError ? automationPanelError(dryRunError, 'Dry-run failed') : null}
      {dryRunResult ? (
        <section className="grid gap-2">
          <h2 className="text-base font-semibold">Dry-run result</h2>
          <JsonDashboardView payload={dryRunResult as unknown as Record<string, unknown>} />
        </section>
      ) : null}

      {error && hasSnapshot ? automationPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
