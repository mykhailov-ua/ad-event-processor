import { useCallback, useState } from 'react';

import {
  addOpsMlLabel,
  getOpsMlModelEval,
  getOpsMlModelStatus,
  listOpsMlLabels,
} from '@/api/ops_api';
import { OpsMlModel } from '@/domains/ops/ops_ml_model';

export function OpsMlModelPage() {
  const [draftIpHash, setDraftIpHash] = useState('');
  const [draftLabel, setDraftLabel] = useState('');
  const [draftReason, setDraftReason] = useState('');
  const [status, setStatus] = useState<Record<string, unknown> | undefined>();
  const [evalBlock, setEvalBlock] = useState<Record<string, unknown> | undefined>();
  const [labels, setLabels] = useState<Awaited<ReturnType<typeof listOpsMlLabels>>>([]);
  const [fetchingStatus, setFetchingStatus] = useState(false);
  const [fetchingEval, setFetchingEval] = useState(false);
  const [fetchingLabels, setFetchingLabels] = useState(false);
  const [savingLabel, setSavingLabel] = useState(false);
  const [statusError, setStatusError] = useState<Error | undefined>();
  const [evalError, setEvalError] = useState<Error | undefined>();
  const [labelsError, setLabelsError] = useState<Error | undefined>();
  const [saveError, setSaveError] = useState<Error | undefined>();
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [hasStatusSnapshot, setHasStatusSnapshot] = useState(false);
  const [hasEvalSnapshot, setHasEvalSnapshot] = useState(false);
  const [hasLabelsSnapshot, setHasLabelsSnapshot] = useState(false);

  const onLoadStatus = useCallback(async () => {
    setFetchingStatus(true);
    setStatusError(undefined);
    try {
      setStatus(await getOpsMlModelStatus());
      setHasStatusSnapshot(true);
    } catch (err) {
      setStatusError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setFetchingStatus(false);
    }
  }, []);

  const onLoadEval = useCallback(async () => {
    setFetchingEval(true);
    setEvalError(undefined);
    try {
      setEvalBlock(await getOpsMlModelEval());
      setHasEvalSnapshot(true);
    } catch (err) {
      setEvalError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setFetchingEval(false);
    }
  }, []);

  const onLoadLabels = useCallback(async () => {
    setFetchingLabels(true);
    setLabelsError(undefined);
    try {
      setLabels(await listOpsMlLabels());
      setHasLabelsSnapshot(true);
    } catch (err) {
      setLabelsError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setFetchingLabels(false);
    }
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
      await onLoadLabels();
    } catch (err) {
      setSaveError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSavingLabel(false);
    }
  }, [draftIpHash, draftLabel, draftReason, onLoadLabels]);

  return (
    <OpsMlModel
      status={status}
      evalBlock={evalBlock}
      labels={labels}
      draftIpHash={draftIpHash}
      draftLabel={draftLabel}
      draftReason={draftReason}
      fetchingStatus={fetchingStatus}
      fetchingEval={fetchingEval}
      fetchingLabels={fetchingLabels}
      savingLabel={savingLabel}
      statusError={statusError}
      evalError={evalError}
      labelsError={labelsError}
      saveError={saveError}
      saveSuccess={saveSuccess}
      hasStatusSnapshot={hasStatusSnapshot}
      hasEvalSnapshot={hasEvalSnapshot}
      hasLabelsSnapshot={hasLabelsSnapshot}
      onDraftIpHashChange={setDraftIpHash}
      onDraftLabelChange={setDraftLabel}
      onDraftReasonChange={setDraftReason}
      onLoadStatus={onLoadStatus}
      onLoadEval={onLoadEval}
      onLoadLabels={onLoadLabels}
      onAddLabel={onAddLabel}
    />
  );
}
