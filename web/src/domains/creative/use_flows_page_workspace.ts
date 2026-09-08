// L3 flows directory: list + visual create; paths validated client-side before POST.
import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { listLanders } from '@/api/landers_api';
import { listOffers } from '@/api/offers_api';
import { createFlow, listFlows } from '@/api/flows_api';
import { buildFlowBodyFromVisual } from '@/domains/creative/flow_editor_form';
import { newFlowPathRow, type FlowPathVisualRow } from '@/domains/creative/flow_path_model';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

export function useFlowsPageWorkspace() {
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const { data, error, fetching } = useResource((signal) => listFlows(signal), [refreshToken]);
  const { data: landers } = useResource((signal) => listLanders(signal), [refreshToken]);
  const { data: offers } = useResource((signal) => listOffers(signal), [refreshToken]);

  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, fetching);

  const [draftName, setDraftName] = useState('');
  const [draftRows, setDraftRows] = useState<FlowPathVisualRow[]>([newFlowPathRow()]);
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | undefined>();
  const [createSuccess, setCreateSuccess] = useState(false);

  const onCreateFlow = useCallback(async () => {
    if (creating) {
      return;
    }
    setCreating(true);
    setCreateError(undefined);
    setCreateSuccess(false);
    try {
      const body = buildFlowBodyFromVisual(draftName, draftRows);
      if (!body.ok) {
        throw new Error(body.error);
      }
      await createFlow(body.body);
      setCreateSuccess(true);
      setDraftName('');
      setDraftRows([newFlowPathRow()]);
      toast.success('Flow created');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setCreateError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [bumpRefreshCoalesced, creating, draftName, draftRows]);

  return {
    items: data,
    fetching,
    error,
    hasSnapshot: data != null,
    landers: landers ?? [],
    offers: offers ?? [],
    draftName,
    draftRows,
    creating,
    createError,
    createSuccess,
    onDraftNameChange: setDraftName,
    onDraftRowsChange: setDraftRows,
    onCreateFlow: () => {
      void onCreateFlow();
    },
  };
}
