// L3 flow editor: GET snapshot keyed by refreshToken; visual path draft until PUT save.
import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { toast } from 'sonner';

import { listLanders } from '@/api/landers_api';
import { listOffers } from '@/api/offers_api';
import { cloneFlow, deleteFlow, getFlow, updateFlow } from '@/api/flows_api';
import {
  buildFlowBodyFromVisual,
  flowVisualRowsFromSnapshot,
} from '@/domains/creative/flow_editor_form';
import type { FlowPathVisualRow } from '@/domains/creative/flow_path_model';
import { newFlowPathRow } from '@/domains/creative/flow_path_model';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { confirmDestructiveAction, mutationError } from '@/lib/mutation_audit';
import { useBreadcrumbSegmentLabel } from '@/shell/breadcrumb_context';
import { useResource } from '@/api/use_resource';

export function useFlowDetailPageWorkspace() {
  const { id } = useParams();
  const navigate = useNavigate();
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

  const { data: landers } = useResource((signal) => listLanders(signal), []);
  const { data: offers } = useResource((signal) => listOffers(signal), []);

  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, fetching);

  const [draftName, setDraftName] = useState('');
  const [draftRows, setDraftRows] = useState<FlowPathVisualRow[]>([newFlowPathRow()]);
  const [showAdvancedJson, setShowAdvancedJson] = useState(false);
  const [appliedSnapshotKey, setAppliedSnapshotKey] = useState('');
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<Error | undefined>();
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<Error | undefined>();
  const [cloning, setCloning] = useState(false);

  useEffect(() => {
    setDraftName('');
    setDraftRows([newFlowPathRow()]);
    setAppliedSnapshotKey('');
  }, [flowId]);

  useEffect(() => {
    if (!data || appliedSnapshotKey === snapshotKey) {
      return;
    }
    setDraftName(data.name ?? '');
    setDraftRows(flowVisualRowsFromSnapshot(data.paths));
    setAppliedSnapshotKey(snapshotKey);
  }, [appliedSnapshotKey, data, snapshotKey]);

  const onSaveFlow = useCallback(async () => {
    if (!flowId) {
      return;
    }
    const update = buildFlowBodyFromVisual(draftName, draftRows);
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
  }, [bumpRefreshCoalesced, draftName, draftRows, flowId]);

  const onCloneFlow = useCallback(async () => {
    if (!flowId || cloning) {
      return;
    }
    setCloning(true);
    setSaveError(undefined);
    try {
      const cloned = await cloneFlow(flowId, { name: `${draftName.trim() || 'Flow'} (copy)` });
      toast.success('Flow cloned');
      navigate(`/flows/${cloned.id}`);
    } catch (err: unknown) {
      setSaveError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCloning(false);
    }
  }, [cloning, draftName, flowId, navigate]);

  const onDeleteFlow = useCallback(async () => {
    if (!flowId || deleting) {
      return;
    }
    const flowName = data?.name ?? flowId;
    const confirmed = confirmDestructiveAction(`Delete flow "${flowName}"?`);
    if (!confirmed) {
      return;
    }
    setDeleting(true);
    setDeleteError(undefined);
    try {
      await deleteFlow(flowId);
      toast.success('Flow deleted');
      navigate('/flows');
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setDeleteError(nextError);
      toast.error(nextError.message);
    } finally {
      setDeleting(false);
    }
  }, [data?.name, deleting, flowId, navigate]);

  useBreadcrumbSegmentLabel(flowId || undefined, data?.name);

  return {
    flow: data,
    fetching,
    error,
    hasSnapshot: data != null,
    landers: landers ?? [],
    offers: offers ?? [],
    draftName,
    draftRows,
    showAdvancedJson,
    saving,
    saveError,
    deleting,
    deleteError,
    cloning,
    onDraftNameChange: setDraftName,
    onDraftRowsChange: setDraftRows,
    onShowAdvancedJsonChange: setShowAdvancedJson,
    onSaveFlow: () => {
      void onSaveFlow();
    },
    onCloneFlow: () => {
      void onCloneFlow();
    },
    onDeleteFlow: () => {
      void onDeleteFlow();
    },
  };
}
