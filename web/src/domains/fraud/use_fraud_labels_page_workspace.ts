// fraud labels directory: server pagination via URL limit/offset; single-row and bulk JSON upsert.
// IP hash validated client-side for UX; server remains authoritative (SV-*).
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';

import { bulkUpsertFraudLabels, listFraudLabels, upsertFraudLabel } from '@/api/fraud_api';
import type { FraudManualLabelBulkRequest } from '@/api/types';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { parseListLimit, parseListOffset } from '@/lib/list_query';
import { mutationError } from '@/lib/mutation_audit';
import { newRandomUuid } from '@/lib/uuid';

const IP_HASH_PATTERN = /^[0-9a-fA-F]{32}$/;
const FRAUD_LABELS_BULK_MAX_ROWS = 500;

export type FraudLabelBulkDraftRow = {
  id: string;
  ip_hash: string;
  label: string;
  reason: string;
};

function createBulkDraftRow(): FraudLabelBulkDraftRow {
  return {
    id: newRandomUuid(),
    ip_hash: '',
    label: '1',
    reason: '',
  };
}

export function useFraudLabelsPageWorkspace() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const { session } = useSession();
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<Error | undefined>();
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [bulkRows, setBulkRows] = useState<FraudLabelBulkDraftRow[]>([createBulkDraftRow()]);
  const [bulkSaving, setBulkSaving] = useState(false);
  const [bulkError, setBulkError] = useState<Error | undefined>();
  const [bulkSuccess, setBulkSuccess] = useState(false);
  const [bulkUpserted, setBulkUpserted] = useState<number | undefined>();

  const appliedCustomerId = searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
  const appliedLimit = parseListLimit(searchParams.get('limit'), 100);
  const appliedOffset = parseListOffset(searchParams.get('offset'));

  const [draftCustomerId, setDraftCustomerId] = useState(appliedCustomerId);
  const [draftIpHash, setDraftIpHash] = useState('');
  const [draftLabel, setDraftLabel] = useState('1');
  const [draftReason, setDraftReason] = useState('');

  useEffect(() => {
    setDraftCustomerId(appliedCustomerId);
  }, [appliedCustomerId]);

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
      return listFraudLabels(
        {
          customer_id: appliedCustomerId,
          limit: appliedLimit,
          offset: appliedOffset,
        },
        signal
      );
    },
    [appliedCustomerId, appliedLimit, appliedOffset, refreshToken, shouldFetch]
  );

  const listBusy = fetching || saving || bulkSaving;
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, listBusy);

  const updateQuery = useCallback(
    (patch: { customer_id?: string; limit?: number; offset?: number }) => {
      const next = new URLSearchParams(searchParams);
      const customerId = patch.customer_id ?? appliedCustomerId;
      const limit = patch.limit ?? appliedLimit;
      const offset = patch.offset ?? appliedOffset;

      if (customerId) {
        next.set('customer_id', customerId);
      } else {
        next.delete('customer_id');
      }
      next.set('limit', String(limit));
      next.set('offset', String(Math.max(0, offset)));
      replaceSearchParams(next);
    },
    [appliedCustomerId, appliedLimit, appliedOffset, replaceSearchParams, searchParams]
  );

  const onApplyCustomer = useCallback(() => {
    updateQuery({ customer_id: draftCustomerId.trim(), offset: 0 });
  }, [draftCustomerId, updateQuery]);

  const onPageChange = useCallback(
    (nextOffset: number) => {
      updateQuery({ offset: Math.max(0, nextOffset) });
    },
    [updateQuery]
  );

  const onSaveLabel = useCallback(async () => {
    if (saving) {
      return;
    }
    if (!appliedCustomerId) {
      return;
    }
    const ipHash = draftIpHash.trim();
    if (!IP_HASH_PATTERN.test(ipHash)) {
      setSaveError(new Error('IP hash must be 32 hexadecimal characters'));
      setSaveSuccess(false);
      return;
    }
    const label = Number.parseInt(draftLabel, 10);
    if (label !== 0 && label !== 1) {
      setSaveError(new Error('Label must be 0 or 1'));
      setSaveSuccess(false);
      return;
    }
    setSaving(true);
    setSaveError(undefined);
    setSaveSuccess(false);
    try {
      await upsertFraudLabel(appliedCustomerId, {
        ip_hash: ipHash,
        label,
        reason: draftReason.trim() || undefined,
      });
      setSaveSuccess(true);
      setDraftIpHash('');
      setDraftReason('');
      toast.success('Label saved');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setSaveError(nextError);
      toast.error(nextError.message);
    } finally {
      setSaving(false);
    }
  }, [appliedCustomerId, draftIpHash, draftLabel, draftReason, bumpRefreshCoalesced, saving]);

  const onAddBulkRow = useCallback(() => {
    setBulkRows((rows) => {
      if (rows.length >= FRAUD_LABELS_BULK_MAX_ROWS) {
        return rows;
      }
      return [...rows, createBulkDraftRow()];
    });
  }, []);

  const onRemoveBulkRow = useCallback((rowId: string) => {
    setBulkRows((rows) => {
      const next = rows.filter((row) => row.id !== rowId);
      return next.length > 0 ? next : [createBulkDraftRow()];
    });
  }, []);

  const onBulkRowChange = useCallback(
    (
      rowId: string,
      patch: Partial<Pick<FraudLabelBulkDraftRow, 'ip_hash' | 'label' | 'reason'>>
    ) => {
      setBulkRows((rows) => rows.map((row) => (row.id === rowId ? { ...row, ...patch } : row)));
    },
    []
  );

  const onBulkUpsert = useCallback(async () => {
    if (bulkSaving) {
      return;
    }
    if (!appliedCustomerId) {
      return;
    }
    const rows = bulkRows
      .map((row) => ({
        ip_hash: row.ip_hash.trim(),
        label: Number.parseInt(row.label, 10),
        reason: row.reason.trim() || undefined,
      }))
      .filter((row) => row.ip_hash !== '');
    if (rows.length === 0) {
      setBulkError(new Error('Add at least one row with an IP hash'));
      return;
    }
    if (rows.length > FRAUD_LABELS_BULK_MAX_ROWS) {
      setBulkError(new Error(`Bulk upsert supports at most ${FRAUD_LABELS_BULK_MAX_ROWS} rows`));
      return;
    }
    for (const row of rows) {
      if (!IP_HASH_PATTERN.test(row.ip_hash)) {
        setBulkError(new Error('Each IP hash must be 32 hexadecimal characters'));
        return;
      }
      if (row.label !== 0 && row.label !== 1) {
        setBulkError(new Error('Each label must be 0 or 1'));
        return;
      }
    }
    setBulkSaving(true);
    setBulkError(undefined);
    setBulkSuccess(false);
    setBulkUpserted(undefined);
    try {
      const body: FraudManualLabelBulkRequest = { rows };
      const response = await bulkUpsertFraudLabels(appliedCustomerId, body);
      setBulkSuccess(true);
      setBulkUpserted(response.upserted);
      setBulkRows([createBulkDraftRow()]);
      toast.success(`Bulk upserted ${response.upserted ?? 0} label(s)`);
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setBulkError(nextError);
      toast.error(nextError.message);
    } finally {
      setBulkSaving(false);
    }
  }, [appliedCustomerId, bulkRows, bumpRefreshCoalesced, bulkSaving]);

  return {
    items: data?.items,
    total: data?.total ?? 0,
    limit: data?.limit ?? appliedLimit,
    offset: data?.offset ?? appliedOffset,
    customerId: appliedCustomerId,
    draftCustomerId,
    draftIpHash,
    draftLabel,
    draftReason,
    fetching,
    listRevalidating: listRevalidating || listQueryPending,
    saving,
    error,
    saveError,
    saveSuccess,
    bulkRows,
    bulkSaving,
    bulkError,
    bulkSuccess,
    bulkUpserted,
    bulkMaxRows: FRAUD_LABELS_BULK_MAX_ROWS,
    hasSnapshot: !shouldFetch || data != null,
    onAddBulkRow,
    onRemoveBulkRow,
    onBulkRowChange,
    onBulkUpsert,
    onDraftCustomerIdChange: setDraftCustomerId,
    onDraftIpHashChange: setDraftIpHash,
    onDraftLabelChange: setDraftLabel,
    onDraftReasonChange: setDraftReason,
    onApplyCustomer,
    onPageChange,
    onSaveLabel,
  };
}
