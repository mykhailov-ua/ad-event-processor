import { useMemo } from 'react';

import { listAutomationPresets } from '@/api/automation_api';
import { AutomationPresetsDirectory } from '@/domains/automation/automation_presets_directory';
import { useResource } from '@/api/use_resource';

export function AutomationPresetsPage() {
  const { data, error, fetching } = useResource((signal) => listAutomationPresets(signal), []);

  const items = useMemo(() => data ?? [], [data]);

  return (
    <AutomationPresetsDirectory
      items={items}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
    />
  );
}
