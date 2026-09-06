import { SmartAlertsHistoryDirectory } from '@/domains/automation/smart_alerts_history_directory';
import { useSmartAlertsHistoryPageWorkspace } from '@/domains/automation/use_smart_alerts_history_page_workspace';

export function SmartAlertsHistoryPage() {
  return <SmartAlertsHistoryDirectory {...useSmartAlertsHistoryPageWorkspace()} />;
}
