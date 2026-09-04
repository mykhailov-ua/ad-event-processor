import { listTrafficOptimizerPresets } from '@/api/traffic_optimizer_api';
import { TrafficOptimizerPresetsDirectory } from '@/domains/automation/traffic_optimizer_presets_directory';
import { useResource } from '@/api/use_resource';

export function TrafficOptimizerPresetsPage() {
  const { data, error, fetching } = useResource(
    (signal) => listTrafficOptimizerPresets(signal),
    [],
  );

  return (
    <TrafficOptimizerPresetsDirectory
      items={data}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
    />
  );
}
