import { useCallback, useState } from 'react';

import {
  addOpsMlLabel,
  getOpsMlModelEval,
  getOpsMlModelStatus,
  listOpsMlLabels,
} from '@/api/ops_api';
import { useResource } from '@/api/use_resource';
import { OpsMlModel } from '@/domains/ops/ops_ml_model';

function skipLazyFetch(): Promise<never> {
  return Promise.reject(new DOMException('Skipped', 'AbortError'));
}

export function OpsMlModelPage() {
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
    [statusLoadToken],
  );

  const evalResource = useResource(
    (signal) => {
      if (evalLoadToken === 0) {
        return skipLazyFetch();
      }
      return getOpsMlModelEval(signal);
    },
    [evalLoadToken],
  );

  const labelsResource = useResource(
    (signal) => {
      if (labelsLoadToken === 0) {
        return skipLazyFetch();
      }
      return listOpsMlLabels(signal);
    },
    [labelsLoadToken],
  );

  const onLoadStatus = useCallback(() => {
    setStatusLoadToken((value) => value + 1);
  }, []);

  const onLoadEval = useCallback(() => {
    setEvalLoadToken((value) => value + 1);
  }, []);

  const onLoadLabels = useCallback(() => {
    setLabelsLoadToken((value) => value + 1);
  }, []);

  const onAddLabel = useCallback(async () => {
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
    } catch (err) {
      setSaveError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSavingLabel(false);
    }
  }, [draftIpHash, draftLabel, draftReason]);

  return (
    <OpsMlModel
      status={statusResource.data}
      evalBlock={evalResource.data}
      labels={labelsResource.data ?? []}
      draftIpHash={draftIpHash}
      draftLabel={draftLabel}
      draftReason={draftReason}
      fetchingStatus={statusResource.fetching}
      fetchingEval={evalResource.fetching}
      fetchingLabels={labelsResource.fetching}
      savingLabel={savingLabel}
      statusError={statusResource.error}
      evalError={evalResource.error}
      labelsError={labelsResource.error}
      saveError={saveError}
      saveSuccess={saveSuccess}
      hasStatusSnapshot={statusResource.data != null}
      hasEvalSnapshot={evalResource.data != null}
      hasLabelsSnapshot={labelsResource.data != null}
      onDraftIpHashChange={setDraftIpHash}
      onDraftLabelChange={setDraftLabel}
      onDraftReasonChange={setDraftReason}
      onLoadStatus={onLoadStatus}
      onLoadEval={onLoadEval}
      onLoadLabels={onLoadLabels}
      onAddLabel={() => {
        void onAddLabel();
      }}
    />
  );
}
