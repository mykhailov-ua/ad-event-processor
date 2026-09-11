import { IntegrationsGoogleSheets } from '@/domains/integrations/integrations_google_sheets';
import { useIntegrationsGoogleSheetsPageWorkspace } from '@/domains/integrations/use_integrations_google_sheets_page_workspace';

export function IntegrationsGoogleSheetsPage() {
  return <IntegrationsGoogleSheets {...useIntegrationsGoogleSheetsPageWorkspace()} />;
}
