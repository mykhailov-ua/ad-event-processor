import type { SmartAlertEvent, SmartAlertRule } from '@/api/types';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { SmartAlertsDraft } from '@/domains/alerts/use_smart_alerts_page_workspace';
import {
  smartAlertTemplateLabel,
  type SmartAlertRuleTemplate,
} from '@/domains/alerts/smart_alerts_templates';
import { DashboardPanelSection } from '@/domains/dashboards/dashboard_panel_section';
import { adminTypography } from '@/lib/admin_kit';
import { adminSpacing, opsControlPanelClass } from '@/lib/admin_spacing';
import { PageChrome } from '@/shell/page_chrome';
import { PageSectionStack } from '@/shell/page_layout';
import { ErrorBlock } from '@/shell/error_block';
import { FilterField, FilterPanel } from '@/shell/filter_panel';
import { BentoSection } from '@/shell/bento_card';
import {
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import type { AdminValidationError } from '@/lib/admin_validation_error';

export type SmartAlertsPageViewProps = {
  customerId: string;
  onCustomerIdChange: (value: string) => void;
  draft: SmartAlertsDraft;
  onDraftChange: (patch: Partial<SmartAlertsDraft>) => void;
  templateOptions: Array<{ value: SmartAlertRuleTemplate; label: string; hint: string }>;
  rules: SmartAlertRule[];
  rulesError?: Error;
  rulesFetching: boolean;
  history: SmartAlertEvent[];
  historyError?: Error;
  historyFetching: boolean;
  historyPage: number;
  historyPageSize: number;
  onHistoryPageChange: (page: number) => void;
  canManage: boolean;
  saving: boolean;
  ackingEventId?: string;
  formValidationError?: AdminValidationError;
  onSaveRule: () => void;
  onResetDraft: () => void;
  onSelectRule: (rule: SmartAlertRule) => void;
  onDeleteRule: () => void;
  onToggleRule: (rule: SmartAlertRule) => void;
  onAckEvent: (eventId: string) => void;
};

function historyTemplateLabel(metric: string | undefined): string {
  if (!metric) {
    return '';
  }
  if (metric.startsWith('template:')) {
    return smartAlertTemplateLabel(metric.slice('template:'.length));
  }
  return metric;
}

export function SmartAlertsPageView({
  customerId,
  onCustomerIdChange,
  draft,
  onDraftChange,
  templateOptions,
  rules,
  rulesError,
  rulesFetching,
  history,
  historyError,
  historyFetching,
  historyPage,
  historyPageSize,
  onHistoryPageChange,
  canManage,
  saving,
  ackingEventId,
  formValidationError,
  onSaveRule,
  onResetDraft,
  onSelectRule,
  onDeleteRule,
  onToggleRule,
  onAckEvent,
}: SmartAlertsPageViewProps) {
  const selectedTemplateHint =
    templateOptions.find((row) => row.value === draft.template)?.hint ?? '';

  return (
    <PageChrome
      description="Template-based alert rules with webhook delivery. No custom SQL."
      title="Smart alerts"
      controlPanel={
        <div className={opsControlPanelClass}>
          <FilterPanel aria-label="Alert rule form">
            {formValidationError ? (
              <ErrorBlock error={formValidationError} title="Check alert rule fields" />
            ) : null}
            <FilterField htmlFor="smart-alerts-customer-id" label="Customer ID">
              <Input
                id="smart-alerts-customer-id"
                value={customerId}
                onChange={(event) => onCustomerIdChange(event.target.value)}
              />
            </FilterField>
            <FilterField htmlFor="smart-alerts-template" label="Template">
              <Select
                disabled={!canManage || saving}
                value={draft.template || undefined}
                onValueChange={(value) => onDraftChange({ template: value as SmartAlertRuleTemplate })}
              >
                <SelectTrigger id="smart-alerts-template">
                  <SelectValue placeholder="Select template" />
                </SelectTrigger>
                <SelectContent>
                  {templateOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FilterField>
            {selectedTemplateHint ? (
              <p className={adminTypography.bodyMuted}>{selectedTemplateHint}</p>
            ) : null}
            <FilterField htmlFor="smart-alerts-threshold" label="Threshold">
              <Input
                disabled={!canManage || saving}
                id="smart-alerts-threshold"
                inputMode="decimal"
                value={draft.threshold}
                onChange={(event) => onDraftChange({ threshold: event.target.value })}
              />
            </FilterField>
            <FilterField htmlFor="smart-alerts-campaign-id" label="Campaign ID (optional)">
              <Input
                disabled={!canManage || saving}
                id="smart-alerts-campaign-id"
                value={draft.campaignId}
                onChange={(event) => onDraftChange({ campaignId: event.target.value })}
              />
            </FilterField>
            <FilterField htmlFor="smart-alerts-webhook-url" label="Webhook URL">
              <Input
                disabled={!canManage || saving}
                id="smart-alerts-webhook-url"
                placeholder="https://hooks.example.com/alerts"
                value={draft.webhookUrl}
                onChange={(event) => onDraftChange({ webhookUrl: event.target.value })}
              />
            </FilterField>
            <FilterField htmlFor="smart-alerts-name" label="Name (optional)">
              <Input
                disabled={!canManage || saving}
                id="smart-alerts-name"
                value={draft.name}
                onChange={(event) => onDraftChange({ name: event.target.value })}
              />
            </FilterField>
            <FilterField htmlFor="smart-alerts-enabled" label="Enabled">
              <Checkbox
                checked={draft.enabled}
                disabled={!canManage || saving}
                id="smart-alerts-enabled"
                onCheckedChange={(checked) => onDraftChange({ enabled: checked === true })}
              />
            </FilterField>
            <div className={adminSpacing.flex.buttonGroup}>
              <Button disabled={!canManage || saving} type="button" onClick={onSaveRule}>
                {saving ? 'Saving...' : 'Create / update'}
              </Button>
              <Button disabled={saving} type="button" variant="outline" onClick={onResetDraft}>
                Reset
              </Button>
              <Button
                disabled={!canManage || saving}
                type="button"
                variant="destructive"
                onClick={onDeleteRule}
              >
                Delete
              </Button>
            </div>
          </FilterPanel>
        </div>
      }
    >
      <PageSectionStack>
        {rulesError ? <ErrorBlock error={rulesError} title="Rules load failed" /> : null}
        {historyError ? <ErrorBlock error={historyError} title="History load failed" /> : null}
        {!canManage ? (
          <p className={adminTypography.bodyMuted}>
            Read-only: create, update, delete, and ack require campaigns:write.
          </p>
        ) : null}

        {rulesFetching ? <p className={adminTypography.bodyMuted}>Loading rules...</p> : null}
        {!rulesFetching && rules.length === 0 ? (
          <p className={adminTypography.bodyMuted}>No alert rules for this customer.</p>
        ) : null}
        {rules.length > 0 ? (
          <DashboardPanelSection tableAriaLabel="Smart alert rules" title="Rules">
            <TableHeader>
              <TableRow>
                <DirectoryTableHead>Name</DirectoryTableHead>
                <DirectoryTableHead>Template</DirectoryTableHead>
                <DirectoryTableHead>Threshold</DirectoryTableHead>
                <DirectoryTableHead>Enabled</DirectoryTableHead>
                <DirectoryTableHead>Actions</DirectoryTableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rules.map((rule) => (
                <TableRow key={rule.id}>
                  <TableCell>{rule.name}</TableCell>
                  <TableCell>{smartAlertTemplateLabel(rule.template) || rule.metric}</TableCell>
                  <TableCell>{rule.threshold ?? ''}</TableCell>
                  <TableCell>{rule.enabled ? 'yes' : 'no'}</TableCell>
                  <TableCell>
                    <div
                      aria-label={`Actions for ${rule.id}`}
                      className={adminSpacing.flex.buttonGroup}
                    >
                      <Button type="button" variant="outline" onClick={() => onSelectRule(rule)}>
                        Edit
                      </Button>
                      {canManage ? (
                        <Button
                          disabled={saving}
                          type="button"
                          variant="outline"
                          onClick={() => void onToggleRule(rule)}
                        >
                          {rule.enabled ? 'Disable' : 'Enable'}
                        </Button>
                      ) : null}
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </DashboardPanelSection>
        ) : null}

        <BentoSection title="History">
          {historyFetching ? <p className={adminTypography.bodyMuted}>Loading history...</p> : null}
          {!historyFetching && history.length === 0 ? (
            <p className={adminTypography.bodyMuted}>No alert history for this page.</p>
          ) : null}
          {history.length > 0 ? (
            <>
              <DashboardPanelSection tableAriaLabel="Smart alert history">
                <TableHeader>
                  <TableRow>
                    <DirectoryTableHead>Fired</DirectoryTableHead>
                    <DirectoryTableHead>Template</DirectoryTableHead>
                    <DirectoryTableHead>Observed</DirectoryTableHead>
                    <DirectoryTableHead>Webhook</DirectoryTableHead>
                    <DirectoryTableHead>Acked</DirectoryTableHead>
                    <DirectoryTableHead>Actions</DirectoryTableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {history.map((event) => (
                    <TableRow key={event.id}>
                      <TableCell>
                        {event.fired_at ? new Date(event.fired_at).toLocaleString() : ''}
                      </TableCell>
                      <TableCell>{historyTemplateLabel(event.metric)}</TableCell>
                      <TableCell>{event.observed_value ?? ''}</TableCell>
                      <TableCell>{event.webhook_status ?? ''}</TableCell>
                      <TableCell>{event.acked_at ? 'yes' : 'no'}</TableCell>
                      <TableCell>
                        {event.acked_at || !event.id ? null : (
                          <Button
                            disabled={!canManage || ackingEventId === event.id}
                            type="button"
                            variant="outline"
                            onClick={() => onAckEvent(event.id!)}
                          >
                            {ackingEventId === event.id ? 'Acking...' : 'Ack'}
                          </Button>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </DashboardPanelSection>
              <div className={adminSpacing.flex.buttonGroup}>
                <Button
                  disabled={historyPage <= 0 || historyFetching}
                  type="button"
                  variant="outline"
                  onClick={() => onHistoryPageChange(historyPage - 1)}
                >
                  Previous
                </Button>
                <span className={adminTypography.bodyMuted}>Page {historyPage + 1}</span>
                <Button
                  disabled={history.length < historyPageSize || historyFetching}
                  type="button"
                  variant="outline"
                  onClick={() => onHistoryPageChange(historyPage + 1)}
                >
                  Next
                </Button>
              </div>
            </>
          ) : null}
        </BentoSection>
      </PageSectionStack>
    </PageChrome>
  );
}
