// L3 sellers.json rows: inline edit + create/delete; list refresh coalesced while saving.
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';

import {
  createSupplySeller,
  deleteSupplySeller,
  listSupplySellers,
  updateSupplySeller,
} from '@/api/supply_api';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

type SellerEditRow = {
  seller_id: string;
  domain: string;
  seller_type: string;
  name: string;
};

export function useSupplySellersPageWorkspace() {
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const { data, error, fetching } = useResource(
    (signal) => listSupplySellers(signal),
    [refreshToken]
  );

  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, fetching);

  const [draftSellerId, setDraftSellerId] = useState('');
  const [draftDomain, setDraftDomain] = useState('');
  const [draftSellerType, setDraftSellerType] = useState('');
  const [draftName, setDraftName] = useState('');
  const [editRows, setEditRows] = useState<Record<number, SellerEditRow>>({});
  const [acting, setActing] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>(undefined);
  const [createSuccess, setCreateSuccess] = useState(false);

  useEffect(() => {
    if (!data?.length) {
      return;
    }
    const next: Record<number, SellerEditRow> = {};
    for (const row of data) {
      next[row.id] = {
        seller_id: row.seller_id,
        domain: row.domain,
        seller_type: row.seller_type,
        name: row.name,
      };
    }
    setEditRows(next);
  }, [data]);

  const bumpReload = bumpRefreshCoalesced;

  const onCreateSeller = useCallback(() => {
    if (
      !draftSellerId.trim() ||
      !draftDomain.trim() ||
      !draftSellerType.trim() ||
      !draftName.trim()
    ) {
      setActionError(new Error('Seller ID, domain, type, and name are required'));
      return;
    }
    setActing(true);
    setActionError(undefined);
    setCreateSuccess(false);
    void createSupplySeller({
      seller_id: draftSellerId.trim(),
      domain: draftDomain.trim(),
      seller_type: draftSellerType.trim(),
      name: draftName.trim(),
    })
      .then(() => {
        setDraftSellerId('');
        setDraftDomain('');
        setDraftSellerType('');
        setDraftName('');
        setCreateSuccess(true);
        toast.success('Seller created');
        bumpReload();
      })
      .catch((err: unknown) => {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setActing(false);
      });
  }, [bumpReload, draftDomain, draftName, draftSellerId, draftSellerType]);

  const onUpdateSeller = useCallback(
    (id: number) => {
      const edit = editRows[id];
      if (!edit) {
        return;
      }
      setActing(true);
      setActionError(undefined);
      void updateSupplySeller(id, {
        seller_id: edit.seller_id.trim(),
        domain: edit.domain.trim(),
        seller_type: edit.seller_type.trim(),
        name: edit.name.trim(),
      })
        .then(() => {
          bumpReload();
        })
        .catch((err: unknown) => {
          setActionError(err instanceof Error ? err : new Error(String(err)));
        })
        .finally(() => {
          setActing(false);
        });
    },
    [bumpReload, editRows]
  );

  const onDeleteSeller = useCallback(
    (id: number) => {
      setActing(true);
      setActionError(undefined);
      void deleteSupplySeller(id)
        .then(() => {
          bumpReload();
        })
        .catch((err: unknown) => {
          setActionError(err instanceof Error ? err : new Error(String(err)));
        })
        .finally(() => {
          setActing(false);
        });
    },
    [bumpReload]
  );

  const onEditRowChange = useCallback((id: number, field: keyof SellerEditRow, value: string) => {
    setEditRows((prev) => ({
      ...prev,
      [id]: {
        ...prev[id],
        [field]: value,
      },
    }));
  }, []);

  return {
    items: data,
    fetching,
    error,
    hasSnapshot: data != null,
    draftSellerId,
    draftDomain,
    draftSellerType,
    draftName,
    editRows,
    acting,
    actionError,
    createSuccess,
    onDraftSellerIdChange: setDraftSellerId,
    onDraftDomainChange: setDraftDomain,
    onDraftSellerTypeChange: setDraftSellerType,
    onDraftNameChange: setDraftName,
    onEditRowChange,
    onCreateSeller,
    onUpdateSeller,
    onDeleteSeller,
  };
}
