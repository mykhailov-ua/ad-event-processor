import { Link2, Plug, ScrollText, Share2, Tags } from 'lucide-react';

import { BentoSection } from '@/components/system/bento_card';
import { HubLinkCard, HubLinkGrid } from '@/components/system/hub_link_card';
import { PageChrome } from '@/components/system/page_chrome';
import { IntegrationsNav } from '@/domains/integrations/integrations_nav';

const INTEGRATION_LINKS = [
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
    description: 'Conversion postback configs, DLQ retries, and per-campaign delivery status.',
    icon: Plug,
    meta: 'Delivery pipeline',
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
