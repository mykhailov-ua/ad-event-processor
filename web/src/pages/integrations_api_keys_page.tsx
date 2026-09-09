import { IntegrationsApiKeys } from '@/domains/integrations/integrations_api_keys';
import { useIntegrationsApiKeysPageWorkspace } from '@/domains/integrations/use_integrations_api_keys_page_workspace';

export function IntegrationsApiKeysPage() {
  return <IntegrationsApiKeys {...useIntegrationsApiKeysPageWorkspace()} />;
}
