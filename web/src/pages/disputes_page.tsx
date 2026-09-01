import { useMemo } from 'react';

import { listDisputes } from '@/api/platform_api';
import { DisputesDirectory } from '@/domains/platform/disputes_directory';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useResource } from '@/hooks/use_resource';

export function DisputesPage() {
  const {
    appliedCustomerId,
    draftCustomerId,
    setDraftCustomerId,
    applyCustomerScope,
  } = useCustomerScope();

  const { data, error, fetching } = useResource(
    (signal) =>
      listDisputes(
        {
          customer_id: appliedCustomerId || undefined,
          limit: 50,
        },
        signal,
      ),
    [appliedCustomerId],
  );

  const disputes = useMemo(() => data?.disputes ?? [], [data]);

  return (
    <DisputesDirectory
      disputes={disputes}
      appliedCustomerId={appliedCustomerId}
      draftCustomerId={draftCustomerId}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null || Boolean(error)}
      onDraftCustomerIdChange={setDraftCustomerId}
      onApplyCustomerScope={applyCustomerScope}
    />
  );
}
