import { AutomationPresetsDirectory } from '@/domains/automation/automation_presets_directory';
import { useAutomationPresetsPageWorkspace } from '@/domains/automation/use_automation_presets_page_workspace';

export function AutomationPresetsPage() {
  return <AutomationPresetsDirectory {...useAutomationPresetsPageWorkspace()} />;
}
