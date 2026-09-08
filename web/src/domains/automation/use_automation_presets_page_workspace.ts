// automation preset catalog: read-only GET on mount.
import { listAutomationPresets } from '@/api/automation_api';
import { useResource } from '@/api/use_resource';

export function useAutomationPresetsPageWorkspace() {
  const { data, error, fetching } = useResource((signal) => listAutomationPresets(signal), []);

  return {
    items: data,
    fetching,
    error,
    hasSnapshot: data != null,
  };
}
