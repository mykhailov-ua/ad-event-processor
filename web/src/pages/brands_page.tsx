import { useCallback, useMemo, useState } from 'react';
import { toast } from 'sonner';

import { createBrand, listBrands } from '@/api/brands_api';
import { BrandsDirectory } from '@/domains/creative/brands_directory';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useResource } from '@/hooks/use_resource';

export function BrandsPage() {
  const {
    appliedCustomerId,
    draftCustomerId,
    setDraftCustomerId,
    applyCustomerScope,
  } = useCustomerScope();

  const [reloadToken, setReloadToken] = useState(0);
  const shouldFetch = Boolean(appliedCustomerId);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve([]);
      }
      return listBrands({ customer_id: appliedCustomerId }, signal);
    },
    [appliedCustomerId, reloadToken, shouldFetch],
  );

  const [draftBrandName, setDraftBrandName] = useState('');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | undefined>();
  const [createSuccess, setCreateSuccess] = useState(false);

  const items = useMemo(() => data ?? [], [data]);

  const onCreateBrand = useCallback(async () => {
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
      setReloadToken((value) => value + 1);
    } catch (err) {
      setCreateError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [appliedCustomerId, draftBrandName]);

  return (
    <BrandsDirectory
      items={items}
      appliedCustomerId={appliedCustomerId}
      draftCustomerId={draftCustomerId}
      fetching={fetching}
      error={error}
      hasSnapshot={!shouldFetch || data != null}
      onDraftCustomerIdChange={setDraftCustomerId}
      onApplyCustomerScope={applyCustomerScope}
      draftBrandName={draftBrandName}
      creating={creating}
      createError={createError}
      createSuccess={createSuccess}
      onDraftBrandNameChange={setDraftBrandName}
      onCreateBrand={() => {
        void onCreateBrand();
      }}
    />
  );
}
