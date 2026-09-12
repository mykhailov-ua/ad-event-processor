import { Link } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { TrafficOptimizerRule } from '@/api/types';
import {
  IntegrationsPageWithLoad,
  integrationsPanelError,
} from '@/domains/integrations/integrations_nav';
import type { useIntegrationsTrafficOptimizerPageWorkspace } from '@/domains/integrations/use_integrations_traffic_optimizer_page_workspace';
import { CustomerScopeBar } from '@/shell/customer_scope_bar';
import { DirectoryFilterForm, FilterField, FilterPanel } from '@/shell/filter_panel';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { TableHost } from '@/shell/ui_bands';

export type IntegrationsTrafficOptimizerProps = ReturnType<
  typeof useIntegrationsTrafficOptimizerPageWorkspace
>;

export function IntegrationsTrafficOptimizer(workspace: IntegrationsTrafficOptimizerProps) {
  const {
    canWrite,
    appliedCustomerId,
    draftCustomerId,
    onDraftCustomerIdChange,
    onApplyCustomerScope,
    presets,
    rules,
    fetching,
    error,
    hasSnapshot,
    draftPresetKey,
    onDraftPresetKeyChange,
    draftFlowId,
    onDraftFlowIdChange,
    draftCampaignId,
    onDraftCampaignIdChange,
    creating,
    createError,
    createValidationError,
    onCreateRule,
    activeRule,
    dryRunResult,
    dryRunningRuleId,
    dryRunError,
    onDryRunRule,
    applyingRuleId,
    applyError,
    onApplyRule,
    deletingRuleId,
    deleteError,
    onDeleteRule,
  } = workspace;

  const mutationAlerts = (
    <>
      {createError ? integrationsPanelError(createError, 'Create rule failed') : null}
      {dryRunError ? integrationsPanelError(dryRunError, 'Dry-run failed') : null}
      {applyError ? <ErrorBlock error={applyError} title="Apply failed" /> : null}
      {deleteError ? integrationsPanelError(deleteError, 'Delete failed') : null}
    </>
  );

  return (
    <IntegrationsPageWithLoad
      alerts={mutationAlerts}
      blockingErrorTitle="Could not load traffic optimizer"
      fetchState={{ error, fetching, hasSnapshot }}
      header={
        <CustomerScopeBar
          appliedCustomerId={appliedCustomerId}
          draftCustomerId={draftCustomerId}
          onApply={onApplyCustomerScope}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
        />
      }
      title="Traffic optimizer"
    >
      <FilterPanel>
        <h2>Create rule</h2>
        <p>
          Pick a bundled preset and target flow. Dry-run previews server-proposed weights; apply
          runs the same transactional path as the background worker (records optimizer fires).
        </p>
        {createValidationError ? (
          <ErrorBlock error={createValidationError} title="Check create fields" />
        ) : null}
        <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
          <FilterField htmlFor="traffic-opt-preset" label="Preset">
            <Select value={draftPresetKey} onValueChange={onDraftPresetKeyChange}>
              <SelectTrigger id="traffic-opt-preset">
                <SelectValue placeholder="Select preset" />
              </SelectTrigger>
              <SelectContent>
                {presets.map((preset) => (
                  <SelectItem key={preset.key ?? ''} value={preset.key ?? ''}>
                    {preset.title ?? preset.key}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FilterField>
          <FilterField htmlFor="traffic-opt-flow-id" label="Flow ID">
            <Input
              id="traffic-opt-flow-id"
              value={draftFlowId}
              onChange={(event) => onDraftFlowIdChange(event.target.value)}
              placeholder="Flow UUID to optimize"
            />
          </FilterField>
          <FilterField htmlFor="traffic-opt-campaign-id" label="Campaign ID (optional)">
            <Input
              id="traffic-opt-campaign-id"
              value={draftCampaignId}
              onChange={(event) => onDraftCampaignIdChange(event.target.value)}
              placeholder="Campaign UUID filter"
            />
          </FilterField>
          <div className="flex items-end">
            <Button disabled={!canWrite || creating || !appliedCustomerId} onClick={onCreateRule}>
              {creating ? 'Creating...' : 'Create rule'}
            </Button>
          </div>
        </DirectoryFilterForm>
        {!canWrite ? (
          <p role="status">Read-only: campaigns:write is required to create or apply rules.</p>
        ) : null}
      </FilterPanel>

      {!appliedCustomerId ? (
        <EmptyState
          description="Apply a customer ID to list optimizer rules."
          title="Customer scope required"
        />
      ) : hasSnapshot && rules.length === 0 ? (
        <EmptyState
          description="Create a rule above to start optimizing flow weights."
          title="No rules"
        />
      ) : (
        <TableHost>
          <DirectoryTable>
            <TableHeader>
              <TableRow>
                <DirectoryTableHead>Name</DirectoryTableHead>
                <DirectoryTableHead>Scope</DirectoryTableHead>
                <DirectoryTableHead>Flow</DirectoryTableHead>
                <DirectoryTableHead>Enabled</DirectoryTableHead>
                <DirectoryTableHead>Actions</DirectoryTableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rules.map((rule) => (
                <TrafficOptimizerRuleRow
                  key={rule.id}
                  applying={applyingRuleId === rule.id}
                  canWrite={canWrite}
                  deleting={deletingRuleId === rule.id}
                  dryRunning={dryRunningRuleId === rule.id}
                  rule={rule}
                  onApply={() => onApplyRule(rule)}
                  onDelete={() => onDeleteRule(rule)}
                  onDryRun={() => onDryRunRule(rule)}
                />
              ))}
            </TableBody>
          </DirectoryTable>
        </TableHost>
      )}

      {activeRule && dryRunResult ? (
        <FilterPanel>
          <h2>Dry-run: {activeRule.name}</h2>
          {dryRunResult.stale_weights ? (
            <p role="status">Weights may be stale relative to the latest catalog snapshot.</p>
          ) : null}
          {(dryRunResult.arms ?? []).length === 0 ? (
            <p role="status">No arm changes proposed for the current lookback window.</p>
          ) : (
            <DirectoryTable>
              <TableHeader>
                <TableRow>
                  <DirectoryTableHead>Entity</DirectoryTableHead>
                  <DirectoryTableHead>Current</DirectoryTableHead>
                  <DirectoryTableHead>Proposed</DirectoryTableHead>
                  <DirectoryTableHead>Observed</DirectoryTableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(dryRunResult.arms ?? []).map((arm) => (
                  <TableRow key={arm.entity_id}>
                    <TableCell>{arm.entity_id}</TableCell>
                    <TableCell>{arm.current_weight}</TableCell>
                    <TableCell>{arm.proposed_weight}</TableCell>
                    <TableCell>{arm.observed_value}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </DirectoryTable>
          )}
        </FilterPanel>
      ) : null}
    </IntegrationsPageWithLoad>
  );
}

type TrafficOptimizerRuleRowProps = {
  rule: TrafficOptimizerRule;
  canWrite: boolean;
  dryRunning: boolean;
  applying: boolean;
  deleting: boolean;
  onDryRun: () => void;
  onApply: () => void;
  onDelete: () => void;
};

function TrafficOptimizerRuleRow({
  rule,
  canWrite,
  dryRunning,
  applying,
  deleting,
  onDryRun,
  onApply,
  onDelete,
}: TrafficOptimizerRuleRowProps) {
  const flowId = rule.flow_id?.trim();
  return (
    <TableRow>
      <TableCell>{rule.name}</TableCell>
      <TableCell>{rule.scope}</TableCell>
      <TableCell>{flowId ? <Link to={`/flows/${flowId}`}>{flowId}</Link> : '-'}</TableCell>
      <TableCell>{rule.enabled ? 'Yes' : 'No'}</TableCell>
      <TableCell>
        <div className="flex flex-wrap gap-2">
          <Button
            disabled={dryRunning || applying || deleting}
            variant="outline"
            onClick={onDryRun}
          >
            {dryRunning ? 'Dry-running...' : 'Dry-run'}
          </Button>
          <Button disabled={!canWrite || dryRunning || applying || deleting} onClick={onApply}>
            {applying ? 'Applying...' : 'Apply rule'}
          </Button>
          <Button
            disabled={!canWrite || dryRunning || applying || deleting}
            variant="destructive"
            onClick={onDelete}
          >
            {deleting ? 'Deleting...' : 'Delete'}
          </Button>
        </div>
      </TableCell>
    </TableRow>
  );
}
