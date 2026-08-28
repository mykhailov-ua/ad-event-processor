import { useCallback } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { parseTelegramDetailTab, type TelegramDetailTab } from '../helpers/telegram_api.js';
import { can } from '../helpers/permissions.js';
import { TelegramDetailView } from '../ui/telegram/telegram_detail.js';
import { ErrorBlock } from '../ui/system/error_block.js';

export function CampaignTelegramPage() {
  const { id } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const permissions = user?.permissions ?? [];
  const campaignId = id ?? '';
  const maskedOnly = can(permissions, 'campaigns:read:masked') && !can(permissions, 'campaigns:read');
  const tab = parseTelegramDetailTab(searchParams.get('tab'));

  const onTabChange = useCallback(
    (next: TelegramDetailTab) => {
      const params = new URLSearchParams(searchParams);
      if (next === 'bots') {
        params.delete('tab');
      } else {
        params.set('tab', next);
      }
      setSearchParams(params, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  if (!campaignId) {
    return <ErrorBlock error={new Error('missing campaign id')} fallbackTitle="Invalid route" />;
  }

  return (
    <TelegramDetailView
      campaignId={campaignId}
      tab={tab}
      maskedOnly={maskedOnly}
      onTabChange={onTabChange}
    />
  );
}
