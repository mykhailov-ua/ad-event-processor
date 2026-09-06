// L3 traffic optimizer preset catalog: read-only GET on mount.
import { listTrafficOptimizerPresets } from '@/api/traffic_optimizer_api';
import { useResource } from '@/api/use_resource';

export function useTrafficOptimizerPresetsPageWorkspace() {
  const { data, error, fetching } = useResource(
    (signal) => listTrafficOptimizerPresets(signal),
    []
  );

  return {
    items: data,
    fetching,
    error,
    hasSnapshot: data != null,
  };
}
