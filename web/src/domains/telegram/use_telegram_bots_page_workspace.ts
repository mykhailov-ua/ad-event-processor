// L3 telegram bots directory: list snapshot + navigate to per-campaign editor by id.
import { useCallback, useState } from 'react';
import { useNavigate } from 'react-router-dom';

import { listTelegramBots } from '@/api/telegram_api';
import { useResource } from '@/api/use_resource';

export function useTelegramBotsPageWorkspace() {
  const navigate = useNavigate();
  const { data, error, fetching, revalidating: listRevalidating } = useResource((signal) => listTelegramBots(signal), []);

  const [draftCampaignId, setDraftCampaignId] = useState('');
  const [actionError, setActionError] = useState<Error | undefined>(undefined);

  const onOpenEditor = useCallback(() => {
    const campaignId = draftCampaignId.trim();
    if (!campaignId) {
      setActionError(new Error('Campaign ID is required'));
      return;
    }
    setActionError(undefined);
    navigate(`/telegram/bots/${encodeURIComponent(campaignId)}`);
  }, [draftCampaignId, navigate]);

  return {
    bots: data,
    fetching,
    listRevalidating,
    error,
    hasSnapshot: data != null || Boolean(error),
    draftCampaignId,
    acting: false,
    actionError,
    onDraftCampaignIdChange: setDraftCampaignId,
    onOpenEditor,
  };
}
