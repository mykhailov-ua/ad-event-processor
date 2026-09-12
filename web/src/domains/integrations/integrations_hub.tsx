import {
  Bug,
  FileSpreadsheet,
  Gauge,
  Key,
  Link2,
  Plug,
  ScrollText,
  Share2,
  ShieldCheck,
  Tags,
} from 'lucide-react';

import { BentoSection } from '@/shell/bento_card';
import { DirectoryPageShell } from '@/shell/directory_page_shell';
import { HubLinkCard, HubLinkGrid } from '@/shell/hub_link_card';
import { IntegrationsNav } from '@/domains/integrations/integrations_nav';
import { opsControlPanelClass } from '@/lib/admin_spacing';

const INTEGRATION_LINKS = [
  {
    path: '/integrations/api-keys',
    title: 'Service accounts',
    description:
      'Mint Bearer API keys for automation (Dolphin, scripts, CAPI) separate from user login.',
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
  {
    path: '/integrations/google-sheets',
    title: 'Google Sheets',
    description: 'Connect Google OAuth to push report exports into spreadsheets from Export Hub.',
    icon: FileSpreadsheet,
    meta: 'Export destination',
  },
  {
    path: '/integrations/traffic-optimizer',
    title: 'Traffic optimizer',
    description:
      'Bandit presets, dry-run weight suggestions, and apply-to-flow for lander/offer splits.',
    icon: Gauge,
    meta: 'Flow weights',
  },
  {
    path: '/integrations/margin-guard',
    title: 'Margin guard',
    description: 'Placement margin policies, activity log, and override removal (Pro+ license).',
    icon: ShieldCheck,
    meta: 'Margin automation',
  },
];

export function IntegrationsHub() {
  return (
    <DirectoryPageShell
      blockingErrorTitle="Integrations unavailable"
      controlPanel={
        <div className={opsControlPanelClass}>
          <IntegrationsNav />
        </div>
      }
      description="Cost sync, postbacks, schemas, and affiliate mapping."
      fetchState={{ fetching: false, error: undefined, hasSnapshot: true }}
      title="Integrations"
    >
      <BentoSection title="Connections">
        <HubLinkGrid>
          {INTEGRATION_LINKS.map((item) => (
            <HubLinkCard key={item.path} {...item} />
          ))}
        </HubLinkGrid>
      </BentoSection>
    </DirectoryPageShell>
  );
}
