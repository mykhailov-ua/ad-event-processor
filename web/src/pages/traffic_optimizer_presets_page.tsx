import { TrafficOptimizerPresetsDirectory } from '@/domains/automation/traffic_optimizer_presets_directory';
import { useTrafficOptimizerPresetsPageWorkspace } from '@/domains/automation/use_traffic_optimizer_presets_page_workspace';

export function TrafficOptimizerPresetsPage() {
  return <TrafficOptimizerPresetsDirectory {...useTrafficOptimizerPresetsPageWorkspace()} />;
}
