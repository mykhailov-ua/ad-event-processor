import { useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';

import {
  createSupplySeller,
  deleteSupplySeller,
  listSupplySellers,
  updateSupplySeller,
} from '@/api/supply_api';
import { SupplySellersDirectory } from '@/domains/creative/supply_sellers';
import { useResource } from '@/hooks/use_resource';

type SellerEditRow = {
  seller_id: string;
  domain: string;
  seller_type: string;
  name: string;
};

export function SupplySellersPage() {
  const [reloadToken, setReloadToken] = useState(0);
  const { data, error, fetching } = useResource(
    (signal) => listSupplySellers(signal),
    [reloadToken],
  );

  const items = useMemo(() => data ?? [], [data]);

  const [draftSellerId, setDraftSellerId] = useState('');
  const [draftDomain, setDraftDomain] = useState('');
  const [draftSellerType, setDraftSellerType] = useState('');
  const [draftName, setDraftName] = useState('');
  const [editRows, setEditRows] = useState<Record<number, SellerEditRow>>({});
  const [acting, setActing] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>(undefined);
  const [createSuccess, setCreateSuccess] = useState(false);

  useEffect(() => {
    const next: Record<number, SellerEditRow> = {};
    for (const row of items) {
      next[row.id] = {
        seller_id: row.seller_id,
        domain: row.domain,
        seller_type: row.seller_type,
        name: row.name,
      };
    }
    setEditRows(next);
  }, [items]);

  const bumpReload = useCallback(() => {
    setReloadToken((value) => value + 1);
  }, []);

  const onCreateSeller = useCallback(() => {
    if (!draftSellerId.trim() || !draftDomain.trim() || !draftSellerType.trim() || !draftName.trim()) {
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
    [bumpReload, editRows],
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
    [bumpReload],
  );

  const onEditRowChange = useCallback(
    (id: number, field: keyof SellerEditRow, value: string) => {
      setEditRows((prev) => ({
        ...prev,
        [id]: {
          ...prev[id],
          [field]: value,
        },
      }));
    },
    [],
  );

  return (
    <SupplySellersDirectory
      items={items}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
      draftSellerId={draftSellerId}
      draftDomain={draftDomain}
      draftSellerType={draftSellerType}
      draftName={draftName}
      editRows={editRows}
      acting={acting}
      actionError={actionError}
      createSuccess={createSuccess}
      onDraftSellerIdChange={setDraftSellerId}
      onDraftDomainChange={setDraftDomain}
      onDraftSellerTypeChange={setDraftSellerType}
      onDraftNameChange={setDraftName}
      onEditRowChange={onEditRowChange}
      onCreateSeller={onCreateSeller}
      onUpdateSeller={onUpdateSeller}
      onDeleteSeller={onDeleteSeller}
    />
  );
}
