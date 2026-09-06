import { TrafficOptimizerRulesDirectory } from '@/domains/automation/traffic_optimizer_rules_directory';
import { useTrafficOptimizerRulesPageWorkspace } from '@/domains/automation/use_traffic_optimizer_rules_page_workspace';

export function TrafficOptimizerRulesPage() {
  return <TrafficOptimizerRulesDirectory {...useTrafficOptimizerRulesPageWorkspace()} />;
}
