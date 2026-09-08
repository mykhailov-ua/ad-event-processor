// saved views CRUD: customer-scoped list + inline spec_json edit rows.
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';

import {
  createSavedView,
  deleteSavedView,
  listSavedViews,
  updateSavedView,
} from '@/api/saved_views_api';
import type { SavedView } from '@/api/types';
import { confirmDestructiveAction, mutationError } from '@/lib/mutation_audit';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

type EditRow = {
  name: string;
  report_key: string;
  spec_json: string;
};

function editRowFromView(row: SavedView): EditRow {
  return {
    name: row.name ?? '',
    report_key: row.report_key ?? '',
    spec_json: row.spec ? JSON.stringify(row.spec) : '',
  };
}

export function useSavedViewsPageWorkspace() {
  const {
    appliedCustomerId,
    draftCustomerId,
    setDraftCustomerId,
    applyCustomerScope,
    listQueryPending,
  } = useCustomerScope();

  const { refreshToken, bumpRefresh } = useRefreshToken();
  const shouldFetch = Boolean(appliedCustomerId);

  const {
    data,
    error,
    fetching,
    revalidating: listRevalidating,
  } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return listSavedViews({ customer_id: appliedCustomerId }, signal);
    },
    [appliedCustomerId, refreshToken, shouldFetch]
  );

  const [draftName, setDraftName] = useState('');
  const [draftReportKey, setDraftReportKey] = useState('');
  const [draftSpecJson, setDraftSpecJson] = useState('{}');
  const [editRows, setEditRows] = useState<Record<string, EditRow>>({});
  const [acting, setActing] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();
  const [createSuccess, setCreateSuccess] = useState(false);

  const listBusy = fetching || acting;
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, listBusy);

  useEffect(() => {
    if (!data?.length) {
      return;
    }
    setEditRows((current) => {
      const next = { ...current };
      for (const row of data) {
        const rowId = row.id ?? '';
        if (!rowId || next[rowId]) {
          continue;
        }
        next[rowId] = editRowFromView(row);
      }
      return next;
    });
  }, [data]);

  const parseSpec = useCallback((raw: string): Record<string, unknown> | undefined => {
    const trimmed = raw.trim();
    if (!trimmed) {
      return undefined;
    }
    const parsed: unknown = JSON.parse(trimmed);
    if (parsed == null || typeof parsed !== 'object' || Array.isArray(parsed)) {
      throw new Error('Spec must be a JSON object');
    }
    return parsed as Record<string, unknown>;
  }, []);

  const onEditRowChange = useCallback(
    (rowId: string, field: 'name' | 'report_key' | 'spec_json', value: string) => {
      setEditRows((current) => ({
        ...current,
        [rowId]: {
          name: current[rowId]?.name ?? '',
          report_key: current[rowId]?.report_key ?? '',
          spec_json: current[rowId]?.spec_json ?? '',
          [field]: value,
        },
      }));
    },
    []
  );

  const onCreateView = useCallback(async () => {
    if (acting) {
      return;
    }
    const customerId = appliedCustomerId.trim();
    const name = draftName.trim();
    const reportKey = draftReportKey.trim();
    if (!customerId || !name || !reportKey) {
      return;
    }
    setActing(true);
    setActionError(undefined);
    setCreateSuccess(false);
    try {
      const spec = parseSpec(draftSpecJson);
      await createSavedView({
        customer_id: customerId,
        name,
        report_key: reportKey,
        spec,
      });
      setDraftName('');
      setDraftReportKey('');
      setDraftSpecJson('{}');
      setCreateSuccess(true);
      toast.success('Saved view created');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setActing(false);
    }
  }, [
    acting,
    appliedCustomerId,
    draftName,
    draftReportKey,
    draftSpecJson,
    parseSpec,
    bumpRefreshCoalesced,
  ]);

  const onUpdateView = useCallback(
    async (rowId: string) => {
      if (acting) {
        return;
      }
      const customerId = appliedCustomerId.trim();
      const edit = editRows[rowId];
      if (!customerId || !edit) {
        return;
      }
      setActing(true);
      setActionError(undefined);
      try {
        const spec = parseSpec(edit.spec_json);
        await updateSavedView(rowId, {
          name: edit.name.trim(),
          report_key: edit.report_key.trim(),
          spec,
        });
        bumpRefreshCoalesced();
      } catch (err: unknown) {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        setActing(false);
      }
    },
    [acting, appliedCustomerId, editRows, parseSpec, bumpRefreshCoalesced]
  );

  const onDeleteView = useCallback(
    async (rowId: string) => {
      if (acting) {
        return;
      }
      const row = data?.find((item) => item.id === rowId);
      const label = editRows[rowId]?.name?.trim() || row?.name?.trim() || rowId;
      if (!confirmDestructiveAction(`Delete saved view "${label}"?`)) {
        return;
      }
      setActing(true);
      setActionError(undefined);
      try {
        await deleteSavedView(rowId);
        setEditRows((current) => {
          const next = { ...current };
          delete next[rowId];
          return next;
        });
        toast.success('Saved view deleted');
        bumpRefreshCoalesced();
      } catch (err: unknown) {
        const nextError = mutationError(err);
        setActionError(nextError);
        toast.error(nextError.message);
      } finally {
        setActing(false);
      }
    },
    [acting, bumpRefreshCoalesced, data, editRows]
  );

  return {
    views: data,
    appliedCustomerId,
    draftCustomerId,
    fetching,
    listRevalidating: listRevalidating || listQueryPending,
    error,
    hasSnapshot: data != null,
    draftName,
    draftReportKey,
    draftSpecJson,
    editRows,
    acting,
    actionError,
    createSuccess,
    onDraftCustomerIdChange: setDraftCustomerId,
    onApplyCustomerScope: applyCustomerScope,
    onDraftNameChange: setDraftName,
    onDraftReportKeyChange: setDraftReportKey,
    onDraftSpecJsonChange: setDraftSpecJson,
    onEditRowChange,
    onCreateView: () => {
      void onCreateView();
    },
    onUpdateView: (rowId: string) => {
      void onUpdateView(rowId);
    },
    onDeleteView: (rowId: string) => {
      void onDeleteView(rowId);
    },
  };
}
