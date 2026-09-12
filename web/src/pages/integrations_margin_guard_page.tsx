import { IntegrationsMarginGuard } from '@/domains/integrations/integrations_margin_guard';
import { useIntegrationsMarginGuardPageWorkspace } from '@/domains/integrations/use_integrations_margin_guard_page_workspace';

export function IntegrationsMarginGuardPage() {
  return <IntegrationsMarginGuard {...useIntegrationsMarginGuardPageWorkspace()} />;
}
