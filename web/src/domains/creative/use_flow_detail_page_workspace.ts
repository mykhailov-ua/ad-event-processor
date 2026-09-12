import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { toast } from 'sonner';

import { listLanders, listOffers } from '@/api/creative_api';
import { isAbortError } from '@/api/client';
import { cloneFlow, deleteFlow, getFlow, updateFlow } from '@/api/flows_api';
import { useResource } from '@/api/use_resource';
import {
  buildFlowBodyFromVisual,
  flowVisualRowsFromSnapshot,
} from '@/domains/creative/flow_editor_form';
import type { FlowPathVisualRow } from '@/domains/creative/flow_path_model';
import { useSession } from '@/hooks/use_session';
import { useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { toError } from '@/lib/admin_error';
import { validationError } from '@/lib/admin_validation_error';
import { useBreadcrumbSegmentLabel } from '@/shell/breadcrumb_context';

export function useFlowDetailPageWorkspace() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { user } = useSession();
  const canWrite = user?.permissions?.includes('campaigns:write') ?? false;
  const { refreshToken, bumpRefresh } = useRefreshToken();

  const [draftName, setDraftName] = useState('');
  const [draftRows, setDraftRows] = useState<FlowPathVisualRow[]>([]);
  const [showAdvancedJson, setShowAdvancedJson] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<Error | undefined>();
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<Error | undefined>();
  const [cloning, setCloning] = useState(false);

  const { data, error, fetching, revalidating } = useResource(
    (signal) => {
      if (!id) {
        return Promise.reject(validationError('Flow id is required.', { field: 'id' }));
      }
      return getFlow(id, signal);
    },
    [id, refreshToken]
  );

  const landersResource = useResource((signal) => listLanders({ limit: 200 }, signal), []);
  const offersResource = useResource((signal) => listOffers(signal), []);

  useEffect(() => {
    if (!data) {
      return;
    }
    setDraftName(data.name ?? '');
    setDraftRows(flowVisualRowsFromSnapshot(data.paths));
  }, [data, refreshToken]);

  useBreadcrumbSegmentLabel(id, data?.name);

  const onSaveFlow = useCallback(async () => {
    if (!id || !canWrite) {
      return;
    }
    const built = buildFlowBodyFromVisual(draftName, draftRows);
    if (!built.ok) {
      setSaveError(new Error(built.error));
      return;
    }
    setSaving(true);
    setSaveError(undefined);
    try {
      await updateFlow(id, built.body);
      toast.success('Flow saved');
      bumpRefresh();
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setSaveError(toError(err));
    } finally {
      setSaving(false);
    }
  }, [bumpRefresh, canWrite, draftName, draftRows, id]);

  const onCloneFlow = useCallback(async () => {
    if (!id || !canWrite) {
      return;
    }
    setCloning(true);
    try {
      const cloned = await cloneFlow(id, { name: `${draftName.trim() || 'Flow'} (copy)` });
      toast.success('Flow cloned');
      navigate(`/flows/${cloned.id}`);
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setSaveError(toError(err));
    } finally {
      setCloning(false);
    }
  }, [canWrite, draftName, id, navigate]);

  const onDeleteFlow = useCallback(async () => {
    if (!id || !canWrite) {
      return;
    }
    setDeleting(true);
    setDeleteError(undefined);
    try {
      await deleteFlow(id);
      toast.success('Flow deleted');
      navigate('/flows');
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setDeleteError(toError(err));
    } finally {
      setDeleting(false);
    }
  }, [canWrite, id, navigate]);

  const hasSnapshot = data != null;

  return {
    flow: data,
    fetching: fetching || revalidating,
    error,
    hasSnapshot,
    landers: landersResource.data ?? [],
    offers: offersResource.data ?? [],
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
    onSaveFlow: canWrite ? onSaveFlow : undefined,
    onCloneFlow: canWrite ? onCloneFlow : undefined,
    onDeleteFlow: canWrite ? onDeleteFlow : undefined,
  };
}

export type FlowDetailPageWorkspace = ReturnType<typeof useFlowDetailPageWorkspace>;
