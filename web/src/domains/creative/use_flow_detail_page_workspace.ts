// L3 flow editor: GET snapshot keyed by refreshToken; local draft until PATCH save.
import { useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { toast } from 'sonner';

import { getFlow, updateFlow } from '@/api/flows_api';
import {
  buildFlowUpdateBody,
  DEFAULT_FLOW_PATHS_JSON,
  flowDraftFromSnapshot,
} from '@/domains/creative/flow_editor_form';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useBreadcrumbSegmentLabel } from '@/shell/breadcrumb_context';
import { useResource } from '@/api/use_resource';

export function useFlowDetailPageWorkspace() {
  const { id } = useParams();
  const flowId = id ?? '';
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const snapshotKey = `${flowId}:${refreshToken}`;

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!flowId) {
        return Promise.reject(new Error('Flow ID required'));
      }
      return getFlow(flowId, signal);
    },
    [flowId, refreshToken]
  );

  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, fetching);

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
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setSaveError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSaving(false);
    }
  }, [bumpRefreshCoalesced, draftName, draftPathsJson, flowId]);

  useBreadcrumbSegmentLabel(flowId || undefined, data?.name);

  return {
    flow: data,
    fetching,
    error,
    hasSnapshot: data != null,
    draftName,
    draftPathsJson,
    saving,
    saveError,
    onDraftNameChange: setDraftName,
    onDraftPathsJsonChange: setDraftPathsJson,
    onSaveFlow: () => {
      void onSaveFlow();
    },
  };
}
