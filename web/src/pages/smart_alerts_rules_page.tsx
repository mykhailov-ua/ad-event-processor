import { SmartAlertsRulesDirectory } from '@/domains/automation/smart_alerts_rules_directory';
import { useSmartAlertsRulesPageWorkspace } from '@/domains/automation/use_smart_alerts_rules_page_workspace';

export function SmartAlertsRulesPage() {
  return <SmartAlertsRulesDirectory {...useSmartAlertsRulesPageWorkspace()} />;
}
