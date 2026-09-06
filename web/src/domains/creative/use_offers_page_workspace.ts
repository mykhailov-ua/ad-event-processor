// L3 offers directory: list + inline create; coalesced list refresh.
import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { createOffer, listOffers } from '@/api/offers_api';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

export function useOffersPageWorkspace() {
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const { data, error, fetching } = useResource((signal) => listOffers(signal), [refreshToken]);

  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, fetching);

  const [draftName, setDraftName] = useState('');
  const [draftUrl, setDraftUrl] = useState('');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<Error | undefined>();
  const [createSuccess, setCreateSuccess] = useState(false);

  const onCreateOffer = useCallback(async () => {
    if (creating) {
      return;
    }
    const name = draftName.trim();
    const url = draftUrl.trim();
    if (!name || !url) {
      setCreateError(new Error('Name and URL are required.'));
      return;
    }
    setCreating(true);
    setCreateError(undefined);
    setCreateSuccess(false);
    try {
      await createOffer({ name, url });
      setCreateSuccess(true);
      setDraftName('');
      setDraftUrl('');
      toast.success('Offer created');
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
    onCreateOffer: () => {
      void onCreateOffer();
    },
  };
}
