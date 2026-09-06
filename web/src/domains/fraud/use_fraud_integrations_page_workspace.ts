// L3 fraud integration health list: scoped by applied customer_id in URL; no fetch until customer applied.
import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { listFraudIntegrations } from '@/api/fraud_api';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';

export function useFraudIntegrationsPageWorkspace() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { session } = useSession();

  const appliedCustomerId = searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
  const [draftCustomerId, setDraftCustomerId] = useState(appliedCustomerId);

  useEffect(() => {
    setDraftCustomerId(appliedCustomerId);
  }, [appliedCustomerId]);

  const shouldFetch = Boolean(appliedCustomerId);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return listFraudIntegrations(appliedCustomerId, signal);
    },
    [appliedCustomerId, shouldFetch]
  );

  const onApplyCustomer = useCallback(() => {
    const next = new URLSearchParams(searchParams);
    const trimmed = draftCustomerId.trim();
    if (trimmed) {
      next.set('customer_id', trimmed);
    } else {
      next.delete('customer_id');
    }
    setSearchParams(next, { replace: true });
  }, [draftCustomerId, searchParams, setSearchParams]);

  return {
    items: data,
    customerId: appliedCustomerId,
    draftCustomerId,
    fetching,
    error,
    hasSnapshot: data != null,
    onDraftCustomerIdChange: setDraftCustomerId,
    onApplyCustomer,
  };
}
