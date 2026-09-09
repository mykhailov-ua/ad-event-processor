import { Bug, Key, Link2, Plug, ScrollText, Share2, Tags } from 'lucide-react';

import { BentoSection } from '@/shell/bento_card';
import { HubLinkCard, HubLinkGrid } from '@/shell/hub_link_card';
import { PageChrome } from '@/shell/page_chrome';
import { IntegrationsNav } from '@/domains/integrations/integrations_nav';

const INTEGRATION_LINKS = [
  {
    path: '/integrations/api-keys',
    title: 'Service accounts',
    description: 'Mint Bearer API keys for automation (Dolphin, scripts, CAPI) separate from user login.',
    icon: Key,
    meta: 'Bearer tokens',
  },
  {
    path: '/integrations/cost-sync',
    title: 'Cost sync',
    description: 'Network credentials, sync history, and manual cost import runs.',
    icon: Share2,
    meta: 'Billing sync',
  },
  {
    path: '/integrations/postbacks',
    title: 'Postbacks',
    description: 'Conversion postback configs and per-campaign delivery status.',
    icon: Plug,
    meta: 'Delivery pipeline',
  },
  {
    path: '/integrations/debugger',
    title: 'Integration debugger',
    description: 'Campaign smoke test, flow validation, and postback dry-run tools.',
    icon: Bug,
    meta: 'Diagnostics',
  },
  {
    path: '/integrations/schemas',
    title: 'Schemas and templates',
    description: 'Integration schema catalog and onboarding template imports.',
    icon: ScrollText,
    meta: 'Schema catalog',
  },
  {
    path: '/integrations/platform-campaigns',
    title: 'Platform campaign links',
    description: 'External ad network campaign IDs linked to internal campaigns.',
    icon: Link2,
    meta: 'ID mapping',
  },
  {
    path: '/integrations/affiliate-presets',
    title: 'Affiliate status presets',
    description: 'Named affiliate conversion status mapping presets.',
    icon: Tags,
    meta: 'Status mapping',
  },
];

export function IntegrationsHub() {
  return (
    <PageChrome
      description="Cost sync, postbacks, schemas, and affiliate mapping."
      title="Integrations"
    >
      <IntegrationsNav />
      <BentoSection title="Connections">
        <HubLinkGrid>
          {INTEGRATION_LINKS.map((item) => (
            <HubLinkCard key={item.path} {...item} />
          ))}
        </HubLinkGrid>
      </BentoSection>
    </PageChrome>
  );
}
