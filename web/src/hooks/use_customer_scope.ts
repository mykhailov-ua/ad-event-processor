import { useCallback, useEffect, useState } from 'react';

import { useSession } from '@/hooks/use_session';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';

export function useCustomerScope() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const { session } = useSession();

  const appliedCustomerId = searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
  const [draftCustomerId, setDraftCustomerId] = useState(appliedCustomerId);

  useEffect(() => {
    setDraftCustomerId(appliedCustomerId);
  }, [appliedCustomerId]);

  const applyCustomerScope = useCallback(() => {
    const next = new URLSearchParams(searchParams);
    const trimmed = draftCustomerId.trim();
    if (trimmed) {
      next.set('customer_id', trimmed);
    } else {
      next.delete('customer_id');
    }
    replaceSearchParams(next);
  }, [draftCustomerId, replaceSearchParams, searchParams]);

  return {
    appliedCustomerId,
    draftCustomerId,
    setDraftCustomerId,
    applyCustomerScope,
    listQueryPending,
  };
}
