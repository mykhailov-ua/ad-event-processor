// brands directory: customer-scoped list with edit/delete row actions.
import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { createBrand, deleteBrand, listBrands, updateBrand } from '@/api/brands_api';
import type { Brand } from '@/api/types';
import { confirmDestructiveAction, mutationError } from '@/lib/mutation_audit';
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

  const [editingBrand, setEditingBrand] = useState<Brand | null>(null);
  const [editBrandName, setEditBrandName] = useState('');
  const [saving, setSaving] = useState(false);
  const [editError, setEditError] = useState<Error | undefined>();

  const [actingBrandId, setActingBrandId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<Error | undefined>();

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

  const onOpenEditBrand = useCallback((brand: Brand) => {
    setEditingBrand(brand);
    setEditBrandName(brand.name);
    setEditError(undefined);
  }, []);

  const onCloseEditBrand = useCallback(() => {
    setEditingBrand(null);
    setEditError(undefined);
  }, []);

  const onSaveBrand = useCallback(async () => {
    if (!editingBrand || saving) {
      return;
    }
    const name = editBrandName.trim();
    if (!name) {
      setEditError(new Error('Brand name is required.'));
      return;
    }
    setSaving(true);
    setEditError(undefined);
    try {
      await updateBrand(editingBrand.id, { name });
      toast.success('Brand updated');
      bumpRefreshCoalesced();
      setEditingBrand(null);
    } catch (err: unknown) {
      setEditError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSaving(false);
    }
  }, [bumpRefreshCoalesced, editBrandName, editingBrand, saving]);

  const onDeleteBrand = useCallback(
    async (brand: Brand) => {
      if (actingBrandId) {
        return;
      }
      const confirmed = confirmDestructiveAction(`Delete brand "${brand.name}"?`);
      if (!confirmed) {
        return;
      }
      setActingBrandId(brand.id);
      setActionError(undefined);
      try {
        await deleteBrand(brand.id);
        toast.success('Brand deleted');
        bumpRefreshCoalesced();
      } catch (err: unknown) {
        const nextError = mutationError(err);
        setActionError(nextError);
        toast.error(nextError.message);
      } finally {
        setActingBrandId(null);
      }
    },
    [actingBrandId, bumpRefreshCoalesced]
  );

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
    editingBrand,
    editBrandName,
    saving,
    editError,
    onOpenEditBrand,
    onCloseEditBrand,
    onEditBrandNameChange: setEditBrandName,
    onSaveBrand: () => {
      void onSaveBrand();
    },
    actingBrandId,
    actionError,
    onDeleteBrand: (brand: Brand) => {
      void onDeleteBrand(brand);
    },
  };
}
