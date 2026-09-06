// L3 affiliate status presets: read-only catalog GET on mount.
import { listAffiliateStatusPresets } from '@/api/integrations_api';
import { useResource } from '@/api/use_resource';

export function useIntegrationsAffiliatePresetsPageWorkspace() {
  const { data, error, fetching } = useResource((signal) => listAffiliateStatusPresets(signal), []);

  return {
    presets: data,
    fetching,
    error,
    hasSnapshot: data != null,
  };
}
