// L3 telegram postbacks CRUD: campaign-scoped list + inline edit rows.
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';

import {
  createTelegramPostback,
  deleteTelegramPostback,
  listTelegramPostbacks,
  testTelegramPostback,
  updateTelegramPostback,
} from '@/api/telegram_api';
import { useCampaignScope } from '@/hooks/use_campaign_scope';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

export function useTelegramPostbacksPageWorkspace() {
  const { appliedCampaignId, draftCampaignId, setDraftCampaignId, applyCampaignScope } =
    useCampaignScope();

  const { refreshToken, bumpRefresh } = useRefreshToken();
  const shouldFetch = Boolean(appliedCampaignId);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return listTelegramPostbacks({ campaign_id: appliedCampaignId }, signal);
    },
    [appliedCampaignId, shouldFetch, refreshToken]
  );

  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, fetching);

  const [draftPostbackUrl, setDraftPostbackUrl] = useState('');
  const [editUrls, setEditUrls] = useState<Record<string, string>>({});
  const [acting, setActing] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>(undefined);
  const [actionMessage, setActionMessage] = useState<string | undefined>(undefined);
  const [createSuccess, setCreateSuccess] = useState(false);

  useEffect(() => {
    if (!data?.length) {
      return;
    }
    const next: Record<string, string> = {};
    for (const row of data) {
      if (row.id) {
        next[row.id] = row.postback_url ?? '';
      }
    }
    setEditUrls(next);
  }, [data]);

  const bumpReload = bumpRefreshCoalesced;

  const onCreatePostback = useCallback(() => {
    const url = draftPostbackUrl.trim();
    if (!appliedCampaignId || !url) {
      setActionError(new Error('Campaign and postback URL are required'));
      return;
    }
    setActing(true);
    setActionError(undefined);
    setActionMessage(undefined);
    setCreateSuccess(false);
    void createTelegramPostback({
      campaign_id: appliedCampaignId,
      postback_url: url,
    })
      .then(() => {
        setDraftPostbackUrl('');
        setCreateSuccess(true);
        setActionMessage('Postback created');
        toast.success('Postback created');
        bumpReload();
      })
      .catch((err: unknown) => {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setActing(false);
      });
  }, [appliedCampaignId, bumpReload, draftPostbackUrl]);

  const onUpdatePostback = useCallback(
    (id: string) => {
      const url = (editUrls[id] ?? '').trim();
      if (!url) {
        setActionError(new Error('Postback URL is required'));
        return;
      }
      setActing(true);
      setActionError(undefined);
      setActionMessage(undefined);
      void updateTelegramPostback(id, { postback_url: url })
        .then(() => {
          setActionMessage('Postback updated');
          bumpReload();
        })
        .catch((err: unknown) => {
          setActionError(err instanceof Error ? err : new Error(String(err)));
        })
        .finally(() => {
          setActing(false);
        });
    },
    [bumpReload, editUrls]
  );

  const onDeletePostback = useCallback(
    (id: string) => {
      setActing(true);
      setActionError(undefined);
      setActionMessage(undefined);
      void deleteTelegramPostback(id)
        .then(() => {
          setActionMessage('Postback deleted');
          bumpReload();
        })
        .catch((err: unknown) => {
          setActionError(err instanceof Error ? err : new Error(String(err)));
        })
        .finally(() => {
          setActing(false);
        });
    },
    [bumpReload]
  );

  const onTestPostback = useCallback((id: string) => {
    setActing(true);
    setActionError(undefined);
    setActionMessage(undefined);
    void testTelegramPostback(id)
      .then(() => {
        setActionMessage('Test postback dispatched');
      })
      .catch((err: unknown) => {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setActing(false);
      });
  }, []);

  const onEditUrlChange = useCallback((id: string, value: string) => {
    setEditUrls((prev) => ({ ...prev, [id]: value }));
  }, []);

  return {
    postbacks: data,
    appliedCampaignId,
    draftCampaignId,
    fetching,
    error,
    hasSnapshot: data != null,
    draftPostbackUrl,
    editUrls,
    acting,
    actionError,
    actionMessage,
    createSuccess,
    onDraftCampaignIdChange: setDraftCampaignId,
    onApplyCampaignScope: applyCampaignScope,
    onDraftPostbackUrlChange: setDraftPostbackUrl,
    onEditUrlChange,
    onCreatePostback,
    onUpdatePostback,
    onDeletePostback,
    onTestPostback,
  };
}
