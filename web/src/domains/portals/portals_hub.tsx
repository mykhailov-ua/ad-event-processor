import { Link } from 'react-router-dom';

import { HubLinkCard, HubLinkGrid } from '@/shell/hub_link_card';
import { PageChrome } from '@/shell/page_chrome';
import { PortalsNav } from '@/domains/portals/portals_nav';
import { hasPortalPermission, type PortalKey } from '@/lib/portal_access';

const PORTAL_LINKS: Array<{
  key: PortalKey;
  path: string;
  title: string;
  description: string;
}> = [
  {
    key: 'selfserve',
    path: '/selfserve',
    title: 'Self-serve',
    description: 'Payment intents, invoices, API keys, pause and resume campaigns.',
  },
  {
    key: 'publisher',
    path: '/publisher/dashboard',
    title: 'Publisher',
    description: 'Scoped supply publisher KPI dashboard and revenue statements.',
  },
  {
    key: 'telegram',
    path: '/telegram',
    title: 'Telegram',
    description: 'Configured Mini App bots and webhook metadata.',
  },
  {
    key: 'reportSchedules',
    path: '/report-schedules',
    title: 'Report schedules',
    description: 'Cron-driven report delivery schedules per customer.',
  },
  {
    key: 'savedViews',
    path: '/views',
    title: 'Saved views',
    description: 'Named report filter specs saved per customer.',
  },
  {
    key: 'forecast',
    path: '/forecast/campaign',
    title: 'Campaign forecast',
    description: 'ClickHouse-backed spend and impression advisory for onboarding.',
  },
];

export type PortalsHubProps = {
  permissions: string[] | undefined;
};

export function PortalsHub({ permissions }: PortalsHubProps) {
  const visibleLinks = PORTAL_LINKS.filter((item) => hasPortalPermission(permissions, item.key));

  return (
    <PageChrome title="Secondary portals">
      <PortalsNav />

      {visibleLinks.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No secondary portal surfaces are available for your session permissions.
        </p>
      ) : (
        <HubLinkGrid>
          {visibleLinks.map(({ key: _portalKey, ...item }) => (
            <HubLinkCard key={item.path} {...item} />
          ))}
        </HubLinkGrid>
      )}
    </PageChrome>
  );
}
