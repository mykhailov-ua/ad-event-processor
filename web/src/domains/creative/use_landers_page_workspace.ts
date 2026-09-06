// L3 landers directory: list + inline create form; toast after successful POST (EH-SI1).
import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { createLander, listLanders } from '@/api/landers_api';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

export function useLandersPageWorkspace() {
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const { data, error, fetching } = useResource((signal) => listLanders(signal), [refreshToken]);

  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, fetching);

  const [draftName, setDraftName] = useState('');
  const [draftUrl, setDraftUrl] = useState('');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | undefined>();
  const [createSuccess, setCreateSuccess] = useState(false);

  const onCreateLander = useCallback(async () => {
    if (creating) {
      return;
    }
    const name = draftName.trim();
    if (!name) {
      setCreateError(new Error('Lander name is required.'));
      return;
    }
    setCreating(true);
    setCreateError(undefined);
    setCreateSuccess(false);
    try {
      await createLander({ name, url: draftUrl.trim() || undefined });
      setCreateSuccess(true);
      setDraftName('');
      setDraftUrl('');
      toast.success('Lander created');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setCreateError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCreating(false);
    }
  }, [bumpRefreshCoalesced, creating, draftName, draftUrl]);

  return {
    items: data,
    fetching,
    error,
    hasSnapshot: data != null,
    draftName,
    draftUrl,
    creating,
    createError,
    createSuccess,
    onDraftNameChange: setDraftName,
    onDraftUrlChange: setDraftUrl,
    onCreateLander: () => {
      void onCreateLander();
    },
  };
}
