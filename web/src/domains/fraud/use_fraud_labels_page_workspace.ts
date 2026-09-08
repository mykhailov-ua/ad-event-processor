// L3 fraud labels directory: server pagination via URL limit/offset; single-row and bulk JSON upsert.
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

const IP_HASH_PATTERN = /^[0-9a-fA-F]{32}$/;

export function useFraudLabelsPageWorkspace() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const { session } = useSession();
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<Error | undefined>();
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [draftBulkJson, setDraftBulkJson] = useState('');
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

  const { data, error, fetching, revalidating: listRevalidating } = useResource(
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

  const onBulkUpsert = useCallback(async () => {
    if (bulkSaving) {
      return;
    }
    if (!appliedCustomerId) {
      return;
    }
    const trimmed = draftBulkJson.trim();
    if (!trimmed) {
      return;
    }
    setBulkSaving(true);
    setBulkError(undefined);
    setBulkSuccess(false);
    setBulkUpserted(undefined);
    try {
      const parsed: unknown = JSON.parse(trimmed);
      if (parsed == null || typeof parsed !== 'object' || Array.isArray(parsed)) {
        throw new Error('Bulk body must be a JSON object with a rows array');
      }
      const body = parsed as FraudManualLabelBulkRequest;
      if (!Array.isArray(body.rows) || body.rows.length === 0) {
        throw new Error('rows must be a non-empty array');
      }
      const response = await bulkUpsertFraudLabels(appliedCustomerId, body);
      setBulkSuccess(true);
      setBulkUpserted(response.upserted);
      setDraftBulkJson('');
      toast.success(`Bulk upserted ${response.upserted ?? 0} label(s)`);
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setBulkError(nextError);
      toast.error(nextError.message);
    } finally {
      setBulkSaving(false);
    }
  }, [appliedCustomerId, draftBulkJson, bumpRefreshCoalesced, bulkSaving]);

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
    draftBulkJson,
    bulkSaving,
    bulkError,
    bulkSuccess,
    bulkUpserted,
    hasSnapshot: !shouldFetch || data != null,
    onDraftBulkJsonChange: setDraftBulkJson,
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
