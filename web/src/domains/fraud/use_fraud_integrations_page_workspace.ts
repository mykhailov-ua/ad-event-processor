// fraud integration health list: scoped by applied customer_id in URL; no fetch until customer applied.
import { listFraudIntegrations } from '@/api/fraud_api';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useResource } from '@/api/use_resource';

export function useFraudIntegrationsPageWorkspace() {
  const {
    appliedCustomerId,
    draftCustomerId,
    setDraftCustomerId,
    applyCustomerScope,
    listQueryPending,
  } = useCustomerScope();

  const shouldFetch = Boolean(appliedCustomerId);

  const {
    data,
    error,
    fetching,
    revalidating: listRevalidating,
  } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return listFraudIntegrations(appliedCustomerId, signal);
    },
    [appliedCustomerId, shouldFetch]
  );

  return {
    items: data,
    customerId: appliedCustomerId,
    draftCustomerId,
    fetching,
    listRevalidating: listRevalidating || listQueryPending,
    error,
    hasSnapshot: data != null,
    onDraftCustomerIdChange: setDraftCustomerId,
    onApplyCustomer: applyCustomerScope,
  };
}
