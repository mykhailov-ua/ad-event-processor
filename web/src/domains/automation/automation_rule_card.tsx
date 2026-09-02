import { Gauge, ToggleLeft } from 'lucide-react';

import { RowActionsMenu } from '@/shell/row_actions_menu';
import { BentoCard, BentoGrid, bentoToneFromKey } from '@/shell/bento_card';
import { Checkbox } from '@/components/ui/checkbox';
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { AutomationRule } from '@/api/types';
import {
  type AutomationRuleEditDraft,
  automationRuleEditFromRow,
} from '@/domains/automation/automation_rule_forms';

export function AutomationRuleCard({
  row,
  draft,
  updating,
  deleting,
  dryRunning,
  onDraftChange,
  onSave,
  onDryRun,
  onDelete,
}: {
  row: AutomationRule;
  draft: AutomationRuleEditDraft;
  updating: boolean;
  deleting: boolean;
  dryRunning: boolean;
  onDraftChange: (patch: Partial<AutomationRuleEditDraft>) => void;
  onSave: () => void;
  onDryRun: () => void;
  onDelete: () => void;
}) {
  const ruleId = row.id ?? '';
  const tone = bentoToneFromKey(ruleId || draft.name || 'rule');

  return (
    <BentoCard
      action={
        <RowActionsMenu ariaLabel="Rule actions" disabled={!ruleId || updating || deleting}>
          <DropdownMenuItem disabled={!ruleId || updating || deleting} onClick={onSave}>
            {updating ? 'Saving...' : 'Save'}
          </DropdownMenuItem>
          <DropdownMenuItem disabled={dryRunning || !ruleId || deleting} onClick={onDryRun}>
            {dryRunning ? 'Running...' : 'Dry-run'}
          </DropdownMenuItem>
          <DropdownMenuItem
            className="text-destructive focus:text-destructive"
            disabled={!ruleId || updating || deleting}
            onClick={onDelete}
          >
            {deleting ? 'Deleting...' : 'Delete'}
          </DropdownMenuItem>
        </RowActionsMenu>
      }
      description={`${draft.metric || 'metric'} ${draft.operator || '?'} ${draft.threshold || '0'}`}
      icon={Gauge}
      meta={
        <span className="inline-flex items-center gap-1.5">
          <ToggleLeft aria-hidden className="size-3.5" />
          {draft.enabled ? 'Enabled' : 'Disabled'}
        </span>
      }
      title={
        <Input
          aria-label={`Name for rule ${ruleId}`}
          className="border-transparent bg-transparent px-0 text-base font-medium shadow-none focus-visible:bg-muted/40"
          value={draft.name}
          onChange={(event) => onDraftChange({ name: event.target.value })}
        />
      }
      tone={tone}
    >
      <div className="grid gap-3 sm:grid-cols-3">
        <div className="grid gap-1.5">
          <Label className="text-xs text-muted-foreground" htmlFor={`rule-metric-${ruleId}`}>
            Metric
          </Label>
          <Input
            id={`rule-metric-${ruleId}`}
            value={draft.metric}
            onChange={(event) => onDraftChange({ metric: event.target.value })}
          />
        </div>
        <div className="grid gap-1.5">
          <Label className="text-xs text-muted-foreground" htmlFor={`rule-operator-${ruleId}`}>
            Operator
          </Label>
          <Input
            id={`rule-operator-${ruleId}`}
            value={draft.operator}
            onChange={(event) => onDraftChange({ operator: event.target.value })}
          />
        </div>
        <div className="grid gap-1.5">
          <Label className="text-xs text-muted-foreground" htmlFor={`rule-threshold-${ruleId}`}>
            Threshold
          </Label>
          <Input
            id={`rule-threshold-${ruleId}`}
            inputMode="decimal"
            value={draft.threshold}
            onChange={(event) => onDraftChange({ threshold: event.target.value })}
          />
        </div>
      </div>
      <div className="flex items-center gap-2 pt-1">
        <Checkbox
          aria-label={`Enabled for rule ${ruleId}`}
          checked={draft.enabled}
          id={`rule-enabled-${ruleId}`}
          onCheckedChange={(checked) => onDraftChange({ enabled: checked === true })}
        />
        <Label className="text-sm font-normal text-muted-foreground" htmlFor={`rule-enabled-${ruleId}`}>
          Run when condition matches
        </Label>
      </div>
    </BentoCard>
  );
}

export function AutomationRulesGrid({
  items,
  ruleDrafts,
  updatingRuleId,
  deletingRuleId,
  dryRunRuleId,
  dryRunning,
  onRuleDraftChange,
  onSaveRule,
  onDeleteRule,
  onDryRun,
}: {
  items: AutomationRule[];
  ruleDrafts: Record<string, AutomationRuleEditDraft>;
  updatingRuleId: string | undefined;
  deletingRuleId: string | undefined;
  dryRunRuleId: string | undefined;
  dryRunning: boolean;
  onRuleDraftChange: (ruleId: string, patch: Partial<AutomationRuleEditDraft>) => void;
  onSaveRule: (ruleId: string) => void;
  onDeleteRule: (ruleId: string) => void;
  onDryRun: (ruleId: string) => void;
}) {
  return (
    <BentoGrid>
      {items.map((row) => {
        const ruleId = row.id ?? '';
        const draft = ruleDrafts[ruleId] ?? automationRuleEditFromRow(row);
        return (
          <AutomationRuleCard
            key={ruleId}
            deleting={deletingRuleId === ruleId}
            draft={draft}
            dryRunning={dryRunning && dryRunRuleId === ruleId}
            onDelete={() => onDeleteRule(ruleId)}
            onDraftChange={(patch) => onRuleDraftChange(ruleId, patch)}
            onDryRun={() => onDryRun(ruleId)}
            onSave={() => onSaveRule(ruleId)}
            row={row}
            updating={updatingRuleId === ruleId}
          />
        );
      })}
    </BentoGrid>
  );
}
