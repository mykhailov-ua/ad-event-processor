import { listAffiliateStatusPresets } from '@/api/integrations_api';
import { IntegrationsAffiliatePresets } from '@/domains/integrations/integrations_affiliate_presets';
import { useResource } from '@/api/use_resource';

export function IntegrationsAffiliatePresetsPage() {
  const { data, error, fetching } = useResource(
    (signal) => listAffiliateStatusPresets(signal),
    [],
  );

  return (
    <IntegrationsAffiliatePresets
      presets={data}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
    />
  );
}
