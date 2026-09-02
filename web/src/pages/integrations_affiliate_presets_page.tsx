import { useMemo } from 'react';

import { listAffiliateStatusPresets } from '@/api/integrations_api';
import { IntegrationsAffiliatePresets } from '@/domains/integrations/integrations_affiliate_presets';
import { useResource } from '@/api/use_resource';

export function IntegrationsAffiliatePresetsPage() {
  const { data, error, fetching } = useResource(
    (signal) => listAffiliateStatusPresets(signal),
    [],
  );

  const presets = useMemo(() => data ?? [], [data]);

  return (
    <IntegrationsAffiliatePresets
      presets={presets}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
    />
  );
}
