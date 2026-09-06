import { AutomationRulesDirectory } from '@/domains/automation/automation_rules_directory';
import { useAutomationRulesPageWorkspace } from '@/domains/automation/use_automation_rules_page_workspace';

export function AutomationRulesPage() {
  return <AutomationRulesDirectory {...useAutomationRulesPageWorkspace()} />;
}
