// L3 flows directory: list + create with paths JSON parsed client-side before POST.
import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { createFlow, listFlows } from '@/api/flows_api';
import type { FlowPath } from '@/api/types';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

export function useFlowsPageWorkspace() {
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const { data, error, fetching } = useResource((signal) => listFlows(signal), [refreshToken]);

  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, fetching);

  const [draftName, setDraftName] = useState('');
  const [draftPathsJson, setDraftPathsJson] = useState(
    '[{"weight":100,"lander_id":"","offer_id":""}]'
  );
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | undefined>();
  const [createSuccess, setCreateSuccess] = useState(false);

  const onCreateFlow = useCallback(async () => {
    if (creating) {
      return;
    }
    const name = draftName.trim();
    if (!name) {
      setCreateError(new Error('Flow name is required.'));
      return;
    }
    setCreating(true);
    setCreateError(undefined);
    setCreateSuccess(false);
    try {
      const parsed: unknown = JSON.parse(draftPathsJson);
      if (!Array.isArray(parsed)) {
        throw new Error('Paths must be a JSON array.');
      }
      await createFlow({ name, paths: parsed as FlowPath[] });
      setCreateSuccess(true);
      setDraftName('');
      toast.success('Flow created');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setCreateError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [bumpRefreshCoalesced, creating, draftName, draftPathsJson]);

  return {
    items: data,
    fetching,
    error,
    hasSnapshot: data != null,
    draftName,
    draftPathsJson,
    creating,
    createError,
    createSuccess,
    onDraftNameChange: setDraftName,
    onDraftPathsJsonChange: setDraftPathsJson,
    onCreateFlow: () => {
      void onCreateFlow();
    },
  };
}
