import { IntegrationsSchemas } from '@/domains/integrations/integrations_schemas';
import { useIntegrationsSchemasPageWorkspace } from '@/domains/integrations/use_integrations_schemas_page_workspace';

export function IntegrationsSchemasPage() {
  return <IntegrationsSchemas {...useIntegrationsSchemasPageWorkspace()} />;
}
