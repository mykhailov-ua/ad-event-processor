import { useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';

import {
  createSupplyAdsTxt,
  deleteSupplyAdsTxt,
  listSupplyAdsTxt,
  updateSupplyAdsTxt,
} from '@/api/supply_api';
import { SupplyAdsTxtDirectory } from '@/domains/creative/supply_ads_txt';
import { useResource } from '@/hooks/use_resource';

type AdsTxtEditRow = {
  domain: string;
  publisher_account_id: string;
  relationship: string;
  sort_order: string;
};

function parseSortOrder(value: string): number | undefined {
  const trimmed = value.trim();
  if (!trimmed) {
    return undefined;
  }
  const parsed = Number(trimmed);
  return Number.isFinite(parsed) ? parsed : undefined;
}

export function SupplyAdsTxtPage() {
  const [reloadToken, setReloadToken] = useState(0);
  const { data, error, fetching } = useResource(
    (signal) => listSupplyAdsTxt(signal),
    [reloadToken],
  );

  const items = useMemo(() => data ?? [], [data]);

  const [draftDomain, setDraftDomain] = useState('');
  const [draftAccountId, setDraftAccountId] = useState('');
  const [draftRelationship, setDraftRelationship] = useState('');
  const [draftSortOrder, setDraftSortOrder] = useState('');
  const [editRows, setEditRows] = useState<Record<number, AdsTxtEditRow>>({});
  const [acting, setActing] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>(undefined);
  const [createSuccess, setCreateSuccess] = useState(false);

  useEffect(() => {
    const next: Record<number, AdsTxtEditRow> = {};
    for (const row of items) {
      next[row.id] = {
        domain: row.domain,
        publisher_account_id: row.publisher_account_id,
        relationship: row.relationship,
        sort_order: String(row.sort_order ?? ''),
      };
    }
    setEditRows(next);
  }, [items]);

  const bumpReload = useCallback(() => {
    setReloadToken((value) => value + 1);
  }, []);

  const onCreateRow = useCallback(() => {
    if (!draftDomain.trim() || !draftAccountId.trim() || !draftRelationship.trim()) {
      setActionError(new Error('Domain, account ID, and relationship are required'));
      return;
    }
    setActing(true);
    setActionError(undefined);
    setCreateSuccess(false);
    const sortOrder = parseSortOrder(draftSortOrder);
    void createSupplyAdsTxt({
      domain: draftDomain.trim(),
      publisher_account_id: draftAccountId.trim(),
      relationship: draftRelationship.trim(),
      sort_order: sortOrder,
    })
      .then(() => {
        setDraftDomain('');
        setDraftAccountId('');
        setDraftRelationship('');
        setDraftSortOrder('');
        setCreateSuccess(true);
        toast.success('ads.txt row created');
        bumpReload();
      })
      .catch((err: unknown) => {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setActing(false);
      });
  }, [bumpReload, draftAccountId, draftDomain, draftRelationship, draftSortOrder]);

  const onUpdateRow = useCallback(
    (id: number) => {
      const edit = editRows[id];
      if (!edit) {
        return;
      }
      setActing(true);
      setActionError(undefined);
      const sortOrder = parseSortOrder(edit.sort_order);
      void updateSupplyAdsTxt(id, {
        domain: edit.domain.trim(),
        publisher_account_id: edit.publisher_account_id.trim(),
        relationship: edit.relationship.trim(),
        sort_order: sortOrder,
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

  const onDeleteRow = useCallback(
    (id: number) => {
      setActing(true);
      setActionError(undefined);
      void deleteSupplyAdsTxt(id)
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
    (id: number, field: keyof AdsTxtEditRow, value: string) => {
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
    <SupplyAdsTxtDirectory
      items={items}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
      draftDomain={draftDomain}
      draftAccountId={draftAccountId}
      draftRelationship={draftRelationship}
      draftSortOrder={draftSortOrder}
      editRows={editRows}
      acting={acting}
      actionError={actionError}
      createSuccess={createSuccess}
      onDraftDomainChange={setDraftDomain}
      onDraftAccountIdChange={setDraftAccountId}
      onDraftRelationshipChange={setDraftRelationship}
      onDraftSortOrderChange={setDraftSortOrder}
      onEditRowChange={onEditRowChange}
      onCreateRow={onCreateRow}
      onUpdateRow={onUpdateRow}
      onDeleteRow={onDeleteRow}
    />
  );
}
