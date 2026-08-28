import { useCallback, useEffect, useState } from 'react';
import {
  buildMlEvalUrl,
  buildMlModelUrl,
  fetchMlLabels,
  postMlLabels,
  type MLEvalReport,
  type MLManualLabel,
  type MLModelStatus,
} from '../helpers/ops_api.js';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { useResource } from '../helpers/use_resource.js';
import { OpsMlModelPanel } from '../ui/ops/ops_ml_model_panel.js';

export function OpsMlModelPage() {
  const { data: status, loading: statusLoading, error: statusError } =
    useResource<MLModelStatus>(buildMlModelUrl());
  const { data: evalReport } = useResource<MLEvalReport>(buildMlEvalUrl());

  const [labels, setLabels] = useState<MLManualLabel[]>([]);
  const [labelsLoading, setLabelsLoading] = useState(true);
  const [labelsReloadToken, setLabelsReloadToken] = useState(0);

  const [ipHash, setIpHash] = useState('');
  const [label, setLabel] = useState('1');
  const [reason, setReason] = useState('');
  const [formBusy, setFormBusy] = useState(false);

  useEffect(() => {
    const ctrl = new AbortController();
    let cancelled = false;
    setLabelsLoading(true);
    void (async () => {
      const [result, err] = await to(fetchMlLabels(ctrl.signal));
      if (cancelled) return;
      if (!err) setLabels(result ?? []);
      setLabelsLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [labelsReloadToken]);

  const onSubmitLabel = useCallback(() => {
    const trimmed = ipHash.trim();
    if (!trimmed) return;
    const labelValue = Number.parseInt(label, 10);
    setFormBusy(true);
    void (async () => {
      try {
        await postMlLabels({
          ip_hash: trimmed,
          label: labelValue,
          reason: reason.trim() || undefined,
        });
        pushToastMessage({ title: 'ML label saved', message: trimmed });
        setIpHash('');
        setLabelsReloadToken((token) => token + 1);
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Save label failed',
          message: err instanceof Error ? err.message : 'Save failed',
        });
      } finally {
        setFormBusy(false);
      }
    })();
  }, [ipHash, label, reason]);

  return (
    <OpsMlModelPanel
      status={status}
      evalReport={evalReport}
      labels={labels}
      loading={statusLoading || labelsLoading}
      error={statusError}
      ipHash={ipHash}
      label={label}
      reason={reason}
      formBusy={formBusy}
      onIpHashChange={setIpHash}
      onLabelChange={setLabel}
      onReasonChange={setReason}
      onSubmitLabel={onSubmitLabel}
    />
  );
}
