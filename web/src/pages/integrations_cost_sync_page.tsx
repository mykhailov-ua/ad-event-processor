import { IntegrationsCostSync } from '@/domains/integrations/integrations_cost_sync';
import { useIntegrationsCostSyncPageWorkspace } from '@/domains/integrations/use_integrations_cost_sync_page_workspace';

export function IntegrationsCostSyncPage() {
  return <IntegrationsCostSync {...useIntegrationsCostSyncPageWorkspace()} />;
}
