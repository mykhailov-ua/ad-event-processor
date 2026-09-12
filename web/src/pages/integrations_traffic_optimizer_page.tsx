import { IntegrationsTrafficOptimizer } from '@/domains/integrations/integrations_traffic_optimizer';
import { useIntegrationsTrafficOptimizerPageWorkspace } from '@/domains/integrations/use_integrations_traffic_optimizer_page_workspace';

export function IntegrationsTrafficOptimizerPage() {
  return <IntegrationsTrafficOptimizer {...useIntegrationsTrafficOptimizerPageWorkspace()} />;
}
