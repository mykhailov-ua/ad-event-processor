import { getOpsConsentProofs } from '@/api/ops_api';
import { OpsConsent } from '@/domains/ops/ops_consent';
import { useResource } from '@/api/use_resource';

export function OpsConsentPage() {
  const { data, error, fetching } = useResource(
    (signal) => getOpsConsentProofs(signal),
    [],
  );

  return (
    <OpsConsent
      payload={data}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
    />
  );
}
