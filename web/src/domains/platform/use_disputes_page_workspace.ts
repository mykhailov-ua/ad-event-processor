// L3 disputes directory: optional customer filter via useCustomerScope.
import { useMemo } from 'react';

import { listDisputes } from '@/api/platform_api';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useResource } from '@/api/use_resource';

export function useDisputesPageWorkspace() {
  const { appliedCustomerId, draftCustomerId, setDraftCustomerId, applyCustomerScope } =
    useCustomerScope();

  const { data, error, fetching } = useResource(
    (signal) =>
      listDisputes(
        {
          customer_id: appliedCustomerId || undefined,
          limit: 50,
        },
        signal
      ),
    [appliedCustomerId]
  );

  const disputes = useMemo(() => data?.disputes ?? [], [data]);

  return {
    disputes,
    appliedCustomerId,
    draftCustomerId,
    fetching,
    error,
    hasSnapshot: data != null || Boolean(error),
    onDraftCustomerIdChange: setDraftCustomerId,
    onApplyCustomerScope: applyCustomerScope,
  };
}
