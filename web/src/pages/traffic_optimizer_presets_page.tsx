import { useMemo } from 'react';

import { listTrafficOptimizerPresets } from '@/api/traffic_optimizer_api';
import { TrafficOptimizerPresetsDirectory } from '@/domains/automation/traffic_optimizer_presets_directory';
import { useResource } from '@/hooks/use_resource';

export function TrafficOptimizerPresetsPage() {
  const { data, error, fetching } = useResource(
    (signal) => listTrafficOptimizerPresets(signal),
    [],
  );

  const items = useMemo(() => data ?? [], [data]);

  return (
    <TrafficOptimizerPresetsDirectory
      items={items}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
    />
  );
}
