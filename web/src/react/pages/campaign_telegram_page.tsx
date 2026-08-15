import { useCallback, useRef } from 'react';
import { useParams } from 'react-router-dom';
import * as auth from '../../helpers/auth.js';
import { can, maskLevel } from '../../helpers/permissions.js';
import { mountCampaignTelegramPanel } from '../../panels/campaign_telegram_panel.js';
import { LegacyPanelHost } from '../components/legacy_panel_host.js';

/**
 * Standalone Telegram config page for a campaign.
 */
export function CampaignTelegramPage() {
  const { id = '' } = useParams();
  const user = auth.getUser();
  const permissions = user?.permissions ?? [];
  const masked = maskLevel(permissions) === 'masked';
  const canWrite = can(permissions, 'campaigns:write');
  const panelRef = useRef<ReturnType<typeof mountCampaignTelegramPanel> | null>(null);

  const mount = useCallback((host: HTMLElement) => {
    panelRef.current = mountCampaignTelegramPanel(host, { campaignId: id, canWrite });
    return panelRef.current;
  }, [id, canWrite]);

  if (masked) {
    return (
      <>
        <div className="page-header">
          <h1 className="page-header__title">Telegram</h1>
        </div>
        <p>Telegram configuration is not available for masked accounts.</p>
        <a href={`/campaigns/${encodeURIComponent(id)}`}>Back to campaign</a>
      </>
    );
  }

  return (
    <>
      <div className="page-header">
        <h1 className="page-header__title">Telegram Mini App</h1>
        <p className="text-muted">
          <a href={`/campaigns/${encodeURIComponent(id)}`}>← Campaign</a>
          {' · '}
          <a href={`/reports/telegram?campaign_id=${encodeURIComponent(id)}`}>Open full analytics</a>
        </p>
      </div>
      <LegacyPanelHost active mount={mount} deps={[id, canWrite]} />
    </>
  );
}
