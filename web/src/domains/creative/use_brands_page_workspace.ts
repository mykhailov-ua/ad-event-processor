// L3 brands directory: customer-scoped list (useCustomerScope); no fetch until customer applied.
import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { createBrand, listBrands } from '@/api/brands_api';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

export function useBrandsPageWorkspace() {
  const { appliedCustomerId, draftCustomerId, setDraftCustomerId, applyCustomerScope } =
    useCustomerScope();

  const { refreshToken, bumpRefresh } = useRefreshToken();
  const shouldFetch = Boolean(appliedCustomerId);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return listBrands({ customer_id: appliedCustomerId }, signal);
    },
    [appliedCustomerId, refreshToken, shouldFetch]
  );

  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, fetching);

  const [draftBrandName, setDraftBrandName] = useState('');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | undefined>();
  const [createSuccess, setCreateSuccess] = useState(false);

  const onCreateBrand = useCallback(async () => {
    if (creating) {
      return;
    }
    const name = draftBrandName.trim();
    if (!appliedCustomerId || !name) {
      setCreateError(new Error('Customer scope and brand name are required.'));
      return;
    }
    setCreating(true);
    setCreateError(undefined);
    setCreateSuccess(false);
    try {
      await createBrand({ customer_id: appliedCustomerId, name });
      setCreateSuccess(true);
      setDraftBrandName('');
      toast.success('Brand created');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setCreateError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [appliedCustomerId, bumpRefreshCoalesced, creating, draftBrandName]);

  return {
    items: data,
    appliedCustomerId,
    draftCustomerId,
    fetching,
    error,
    hasSnapshot: data != null,
    onDraftCustomerIdChange: setDraftCustomerId,
    onApplyCustomerScope: applyCustomerScope,
    draftBrandName,
    creating,
    createError,
    createSuccess,
    onDraftBrandNameChange: setDraftBrandName,
    onCreateBrand: () => {
      void onCreateBrand();
    },
  };
}
