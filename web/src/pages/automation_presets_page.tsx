import { listAutomationPresets } from '@/api/automation_api';
import { AutomationPresetsDirectory } from '@/domains/automation/automation_presets_directory';
import { useResource } from '@/api/use_resource';

export function AutomationPresetsPage() {
  const { data, error, fetching } = useResource((signal) => listAutomationPresets(signal), []);

  return (
    <AutomationPresetsDirectory
      items={data}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
    />
  );
}
