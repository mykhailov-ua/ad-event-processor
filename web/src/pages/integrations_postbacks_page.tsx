import { IntegrationsPostbacks } from '@/domains/integrations/integrations_postbacks';
import { useIntegrationsPostbacksPageWorkspace } from '@/domains/integrations/use_integrations_postbacks_page_workspace';

export function IntegrationsPostbacksPage() {
  return <IntegrationsPostbacks {...useIntegrationsPostbacksPageWorkspace()} />;
}
