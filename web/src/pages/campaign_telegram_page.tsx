import { Link, useParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { can, maskLevel } from '../helpers/permissions.js';
import { CampaignTelegramSection } from '../components/campaign_telegram_section.js';

export function CampaignTelegramPage() {
  const { id = '' } = useParams();
  const user = auth.getUser();
  const permissions = user?.permissions ?? [];
  const masked = maskLevel(permissions) === 'masked';
  const canWrite = can(permissions, 'campaigns:write');

  if (masked) {
    return (
      <>
        <div className="page-header">
          <h1 className="page-header__title">Telegram</h1>
        </div>
        <p>Telegram configuration is not available for masked accounts.</p>
        <Link to={`/campaigns/${encodeURIComponent(id)}`}>Back to campaign</Link>
      </>
    );
  }

  return (
    <>
      <div className="page-header">
        <h1 className="page-header__title">Telegram Mini App</h1>
        <p className="text-muted">
          <Link to={`/campaigns/${encodeURIComponent(id)}`}>{'<-'} Campaign</Link>
          {' , '}
          <Link to={`/reports/telegram?campaign_id=${encodeURIComponent(id)}`}>
            Open full analytics
          </Link>
        </p>
      </div>
      <CampaignTelegramSection campaignId={id} canWrite={canWrite} />
    </>
  );
}
