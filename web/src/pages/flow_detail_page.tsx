import { useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { toast } from 'sonner';

import { getFlow, updateFlow } from '@/api/flows_api';
import {
  buildFlowUpdateBody,
  DEFAULT_FLOW_PATHS_JSON,
  flowDraftFromSnapshot,
} from '@/domains/creative/flow_editor_form';
import { FlowDetail } from '@/domains/creative/flow_detail';
import { useBreadcrumbSegmentLabel } from '@/shell/breadcrumb_context';
import { useResource } from '@/api/use_resource';

export function FlowDetailPage() {
  const { id } = useParams();
  const flowId = id ?? '';
  const [reloadToken, setReloadToken] = useState(0);
  const snapshotKey = `${flowId}:${reloadToken}`;

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!flowId) {
        return Promise.reject(new Error('Flow ID required'));
      }
      return getFlow(flowId, signal);
    },
    [flowId, reloadToken],
  );

  const [draftName, setDraftName] = useState('');
  const [draftPathsJson, setDraftPathsJson] = useState(DEFAULT_FLOW_PATHS_JSON);
  const [appliedSnapshotKey, setAppliedSnapshotKey] = useState('');
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<Error | undefined>();

  useEffect(() => {
    setDraftName('');
    setDraftPathsJson(DEFAULT_FLOW_PATHS_JSON);
    setAppliedSnapshotKey('');
  }, [flowId]);

  useEffect(() => {
    if (!data || appliedSnapshotKey === snapshotKey) {
      return;
    }
    const draft = flowDraftFromSnapshot(data);
    setDraftName(draft.name);
    setDraftPathsJson(draft.pathsJson);
    setAppliedSnapshotKey(snapshotKey);
  }, [appliedSnapshotKey, data, snapshotKey]);

  const onSaveFlow = useCallback(async () => {
    if (!flowId) {
      return;
    }
    const update = buildFlowUpdateBody(draftName, draftPathsJson);
    if (!update.ok) {
      setSaveError(new Error(update.error));
      return;
    }
    setSaving(true);
    setSaveError(undefined);
    try {
      await updateFlow(flowId, update.body);
      toast.success('Flow saved');
      setReloadToken((value) => value + 1);
    } catch (err) {
      setSaveError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSaving(false);
    }
  }, [draftName, draftPathsJson, flowId]);

  useBreadcrumbSegmentLabel(flowId || undefined, data?.name);

  return (
    <FlowDetail
      flow={data}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
      draftName={draftName}
      draftPathsJson={draftPathsJson}
      saving={saving}
      saveError={saveError}
      onDraftNameChange={setDraftName}
      onDraftPathsJsonChange={setDraftPathsJson}
      onSaveFlow={() => {
        void onSaveFlow();
      }}
    />
  );
}
