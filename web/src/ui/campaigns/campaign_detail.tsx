import type { Campaign, CampaignDetailTab } from '../../helpers/campaigns_api.js';
import { visibleCampaignTabs } from '../../helpers/campaigns_api.js';
import { ContextBar } from '../shell/context_bar.js';
import { PageChrome } from '../system/page_chrome.js';
import { TabBar } from '../system/tab_bar.js';
import {
  CampaignConfigPanel,
  CampaignEventsPanel,
  CampaignFraudPanel,
  CampaignOverviewPanel,
  CampaignPostbacksPanel,
  CampaignStatsPanel,
  CampaignToolbar,
} from './campaign_tab_panels.js';
import styles from './campaign_detail.module.css';

export type CampaignDetailProps = {
  campaignId: string;
  campaign: Campaign;
  tab: CampaignDetailTab;
  masked: boolean;
  onTabChange: (tab: CampaignDetailTab) => void;
  onReload: () => void;
};

function shortId(id: string): string {
  return id.length > 12 ? `${id.slice(0, 8)}...` : id;
}

export function CampaignDetail({
  campaignId,
  campaign,
  tab,
  masked,
  onTabChange,
  onReload,
}: CampaignDetailProps) {
  const tabs = visibleCampaignTabs(masked);

  return (
    <div className={styles.root}>
      <ContextBar
        parentLabel="Campaigns"
        parentTo="/campaigns"
        currentLabel={campaign.name ?? campaignId}
      />
      <PageChrome
        title={campaign.name ?? 'Campaign'}
        badge={<span className={styles.mono}>{shortId(campaignId)}</span>}
      />
      <CampaignToolbar campaignId={campaignId} onReload={onReload} />
      <TabBar
        tabs={tabs}
        active={tab}
        onChange={(next) => onTabChange(next as CampaignDetailTab)}
      />
      <div className={styles.panel} role="tabpanel">
        {tab === 'overview' ? <CampaignOverviewPanel campaign={campaign} /> : null}
        {tab === 'config' ? <CampaignConfigPanel campaign={campaign} onSaved={onReload} /> : null}
        {tab === 'fraud' ? <CampaignFraudPanel campaignId={campaignId} /> : null}
        {tab === 'stats' ? <CampaignStatsPanel campaignId={campaignId} /> : null}
        {tab === 'events' ? <CampaignEventsPanel campaignId={campaignId} /> : null}
        {tab === 'postbacks' ? <CampaignPostbacksPanel campaignId={campaignId} /> : null}
      </div>
    </div>
  );
}
