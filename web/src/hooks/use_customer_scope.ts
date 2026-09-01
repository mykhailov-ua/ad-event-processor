import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { useSession } from '@/hooks/use_session';

export function useCustomerScope() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { session } = useSession();

  const appliedCustomerId =
    searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
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
    setSearchParams(next, { replace: true });
  }, [draftCustomerId, searchParams, setSearchParams]);

  return {
    appliedCustomerId,
    draftCustomerId,
    setDraftCustomerId,
    applyCustomerScope,
  };
}
