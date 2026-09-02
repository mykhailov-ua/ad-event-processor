import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { listFraudIntegrations } from '@/api/fraud_api';
import { FraudIntegrations } from '@/domains/fraud/fraud_integrations';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';

export function FraudIntegrationsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { session } = useSession();

  const appliedCustomerId =
    searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
  const [draftCustomerId, setDraftCustomerId] = useState(appliedCustomerId);

  useEffect(() => {
    setDraftCustomerId(appliedCustomerId);
  }, [appliedCustomerId]);

  const shouldFetch = Boolean(appliedCustomerId);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve([]);
      }
      return listFraudIntegrations(appliedCustomerId, signal);
    },
    [appliedCustomerId, shouldFetch],
  );

  const items = useMemo(() => data ?? [], [data]);

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

  return (
    <FraudIntegrations
      items={items}
      customerId={appliedCustomerId}
      draftCustomerId={draftCustomerId}
      fetching={fetching}
      error={error}
      hasSnapshot={!shouldFetch || data != null}
      onDraftCustomerIdChange={setDraftCustomerId}
      onApplyCustomer={onApplyCustomer}
    />
  );
}
