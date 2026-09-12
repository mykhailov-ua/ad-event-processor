import { SmartAlertsPageView } from '@/domains/alerts/smart_alerts_page_view';
import { useSmartAlertsPageWorkspace } from '@/domains/alerts/use_smart_alerts_page_workspace';

export function AlertsPage() {
  const workspace = useSmartAlertsPageWorkspace();
  return (
    <SmartAlertsPageView
      ackingEventId={workspace.ackingEventId}
      canManage={workspace.canManage}
      customerId={workspace.customerId}
      draft={workspace.draft}
      formValidationError={workspace.formValidationError}
      history={workspace.history}
      historyHasSnapshot={workspace.historyHasSnapshot}
      historyError={workspace.historyError}
      historyFetching={workspace.historyFetching}
      historyPage={workspace.historyPage}
      historyPageSize={workspace.historyPageSize}
      rules={workspace.rules}
      rulesHasSnapshot={workspace.rulesHasSnapshot}
      rulesError={workspace.rulesError}
      rulesFetching={workspace.rulesFetching}
      saving={workspace.saving}
      templateOptions={workspace.templateOptions}
      onAckEvent={workspace.onAckEvent}
      onCustomerIdChange={workspace.setCustomerId}
      onDeleteRule={workspace.onDeleteRule}
      onDraftChange={(patch) => workspace.setDraft({ ...workspace.draft, ...patch })}
      onHistoryPageChange={workspace.setHistoryPage}
      onResetDraft={workspace.resetDraft}
      onSaveRule={workspace.onSaveRule}
      onSelectRule={workspace.loadRuleIntoDraft}
      onToggleRule={workspace.onToggleRule}
    />
  );
}
