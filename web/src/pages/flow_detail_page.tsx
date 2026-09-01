import { useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { toast } from 'sonner';

import { getFlow, updateFlow } from '@/api/flows_api';
import type { FlowPath } from '@/api/types';
import { FlowDetail } from '@/domains/creative/flow_detail';
import { useBreadcrumbSegmentLabel } from '@/components/system/breadcrumb_context';
import { useResource } from '@/hooks/use_resource';

function pathsToJson(paths: unknown): string {
  if (Array.isArray(paths)) {
    return JSON.stringify(paths, null, 2);
  }
  return '[{"weight":100,"landers":[],"offers":[]}]';
}

export function FlowDetailPage() {
  const { id } = useParams();
  const flowId = id ?? '';
  const [reloadToken, setReloadToken] = useState(0);

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
  const [draftPathsJson, setDraftPathsJson] = useState(
    '[{"weight":100,"landers":[],"offers":[]}]',
  );
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<Error | undefined>();

  useEffect(() => {
    if (!data) {
      return;
    }
    setDraftName(data.name);
    setDraftPathsJson(pathsToJson(data.paths));
  }, [data]);

  const onSaveFlow = useCallback(async () => {
    const name = draftName.trim();
    if (!flowId || !name) {
      setSaveError(new Error('Flow name is required.'));
      return;
    }
    setSaving(true);
    setSaveError(undefined);
    try {
      const parsed: unknown = JSON.parse(draftPathsJson);
      if (!Array.isArray(parsed)) {
        throw new Error('Paths must be a JSON array.');
      }
      await updateFlow(flowId, { name, paths: parsed as FlowPath[] });
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
