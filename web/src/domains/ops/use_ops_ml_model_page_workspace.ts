// ML model ops console: lazy GET lanes (status/eval/labels) via loadToken + skipLazyFetch AbortError skip.
import { useCallback, useState } from 'react';

import {
  addOpsMlLabel,
  getOpsMlModelEval,
  getOpsMlModelStatus,
  listOpsMlLabels,
} from '@/api/ops_api';
import { useResource } from '@/api/use_resource';
import { toError } from '@/lib/admin_error.ts';
import { useCoalescedBumpRefresh } from '@/hooks/use_coalesced_refresh_token';

function skipLazyFetch(): Promise<never> {
  return Promise.reject(new DOMException('Skipped', 'AbortError'));
}

export function useOpsMlModelPageWorkspace() {
  const [draftIpHash, setDraftIpHash] = useState('');
  const [draftLabel, setDraftLabel] = useState('');
  const [draftReason, setDraftReason] = useState('');
  const [statusLoadToken, setStatusLoadToken] = useState(0);
  const [evalLoadToken, setEvalLoadToken] = useState(0);
  const [labelsLoadToken, setLabelsLoadToken] = useState(0);
  const [savingLabel, setSavingLabel] = useState(false);
  const [saveError, setSaveError] = useState<Error | undefined>();
  const [saveSuccess, setSaveSuccess] = useState(false);

  const statusResource = useResource(
    (signal) => {
      if (statusLoadToken === 0) {
        return skipLazyFetch();
      }
      return getOpsMlModelStatus(signal);
    },
    [statusLoadToken]
  );

  const evalResource = useResource(
    (signal) => {
      if (evalLoadToken === 0) {
        return skipLazyFetch();
      }
      return getOpsMlModelEval(signal);
    },
    [evalLoadToken]
  );

  const labelsResource = useResource(
    (signal) => {
      if (labelsLoadToken === 0) {
        return skipLazyFetch();
      }
      return listOpsMlLabels(signal);
    },
    [labelsLoadToken]
  );

  const onLoadStatus = useCoalescedBumpRefresh(() => {
    setStatusLoadToken((value) => value + 1);
  }, statusResource.fetching);

  const onLoadEval = useCoalescedBumpRefresh(() => {
    setEvalLoadToken((value) => value + 1);
  }, evalResource.fetching);

  const onLoadLabels = useCoalescedBumpRefresh(() => {
    setLabelsLoadToken((value) => value + 1);
  }, labelsResource.fetching);

  const onAddLabel = useCallback(async () => {
    if (savingLabel) {
      return;
    }
    const ipHash = draftIpHash.trim();
    const labelRaw = draftLabel.trim();
    if (!ipHash || !labelRaw) {
      setSaveError(new Error('IP hash and label are required.'));
      return;
    }
    const label = Number.parseInt(labelRaw, 10);
    if (!Number.isFinite(label)) {
      setSaveError(new Error('Label must be an integer.'));
      return;
    }
    setSavingLabel(true);
    setSaveError(undefined);
    setSaveSuccess(false);
    try {
      await addOpsMlLabel({
        ip_hash: ipHash,
        label,
        reason: draftReason.trim() || undefined,
      });
      setSaveSuccess(true);
      setLabelsLoadToken((value) => value + 1);
    } catch (err: unknown) {
      setSaveError(toError(err));
    } finally {
      setSavingLabel(false);
    }
  }, [draftIpHash, draftLabel, draftReason, savingLabel]);

  return {
    status: statusResource.data,
    evalBlock: evalResource.data,
    labels: labelsResource.data,
    draftIpHash,
    draftLabel,
    draftReason,
    fetchingStatus: statusResource.fetching,
    fetchingEval: evalResource.fetching,
    fetchingLabels: labelsResource.fetching,
    savingLabel,
    statusError: statusResource.error,
    evalError: evalResource.error,
    labelsError: labelsResource.error,
    saveError,
    saveSuccess,
    hasStatusSnapshot: statusResource.data != null,
    hasEvalSnapshot: evalResource.data != null,
    hasLabelsSnapshot: labelsResource.data != null,
    onDraftIpHashChange: setDraftIpHash,
    onDraftLabelChange: setDraftLabel,
    onDraftReasonChange: setDraftReason,
    onLoadStatus,
    onLoadEval,
    onLoadLabels,
    onAddLabel: () => {
      void onAddLabel();
    },
  };
}
