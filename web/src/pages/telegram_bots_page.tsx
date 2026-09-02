import { useCallback, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';

import { listTelegramBots } from '@/api/telegram_api';
import { TelegramBotsDirectory } from '@/domains/telegram/bots_directory';
import { useResource } from '@/api/use_resource';

export function TelegramBotsPage() {
  const navigate = useNavigate();
  const { data, error, fetching } = useResource((signal) => listTelegramBots(signal), []);

  const [draftCampaignId, setDraftCampaignId] = useState('');
  const [actionError, setActionError] = useState<Error | undefined>(undefined);

  const bots = useMemo(() => data ?? [], [data]);

  const onOpenEditor = useCallback(() => {
    const campaignId = draftCampaignId.trim();
    if (!campaignId) {
      setActionError(new Error('Campaign ID is required'));
      return;
    }
    setActionError(undefined);
    navigate(`/telegram/bots/${encodeURIComponent(campaignId)}`);
  }, [draftCampaignId, navigate]);

  return (
    <TelegramBotsDirectory
      bots={bots}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null || Boolean(error)}
      draftCampaignId={draftCampaignId}
      acting={false}
      actionError={actionError}
      onDraftCampaignIdChange={setDraftCampaignId}
      onOpenEditor={onOpenEditor}
    />
  );
}
