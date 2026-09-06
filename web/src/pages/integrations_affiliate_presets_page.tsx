import { IntegrationsAffiliatePresets } from '@/domains/integrations/integrations_affiliate_presets';
import { useIntegrationsAffiliatePresetsPageWorkspace } from '@/domains/integrations/use_integrations_affiliate_presets_page_workspace';

export function IntegrationsAffiliatePresetsPage() {
  return <IntegrationsAffiliatePresets {...useIntegrationsAffiliatePresetsPageWorkspace()} />;
}
