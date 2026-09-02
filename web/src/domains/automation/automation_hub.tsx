import {
  Activity,
  Bell,
  Gauge,
  Route,
  Shield,
  SlidersHorizontal,
  Sparkles,
  Workflow,
} from 'lucide-react';

import { BentoSection } from '@/shell/bento_card';
import { HubLinkCard, HubLinkGrid } from '@/shell/hub_link_card';
import { PageChrome } from '@/shell/page_chrome';
import { AutomationNav } from '@/domains/automation/automation_nav';

const CATALOG_LINKS = [
  {
    path: '/automation/presets',
    title: 'Automation presets',
    description: 'Bundled rule templates with parameter schemas for quick rollout.',
    icon: Sparkles,
    meta: 'Template catalog',
  },
  {
    path: '/traffic-optimizer/presets',
    title: 'Traffic optimizer presets',
    description: 'Lander, offer, and creative weight presets for bandit optimization.',
    icon: SlidersHorizontal,
    meta: 'Optimizer templates',
  },
];

const RULES_LINKS = [
  {
    path: '/automation/rules',
    title: 'Automation rules',
    description: 'Customer-scoped rules with server-side dry-run against delivery metrics.',
    icon: Workflow,
    meta: 'Dry-run supported',
  },
  {
    path: '/traffic-optimizer/rules',
    title: 'Traffic optimizer rules',
    description: 'Bandit and proportional rules with proposed weight previews.',
    icon: Route,
    meta: 'Weight proposals',
  },
  {
    path: '/smart-alerts/rules',
    title: 'Smart alert rules',
    description: 'Webhook alert thresholds scoped per customer or campaign.',
    icon: Bell,
    meta: 'Webhook delivery',
  },
];

const MONITORING_LINKS = [
  {
    path: '/smart-alerts/history',
    title: 'Smart alert history',
    description: 'Fired alert events with acknowledge actions and delivery status.',
    icon: Activity,
    meta: 'Event timeline',
  },
  {
    path: '/margin-guard/policies',
    title: 'Margin guard policies',
    description: 'Campaign ROI and spend guard policies with breach thresholds.',
    icon: Shield,
    meta: 'Policy guardrails',
  },
  {
    path: '/margin-guard/activity',
    title: 'Margin guard activity',
    description: 'Recent margin guard actions and enforcement outcomes per campaign.',
    icon: Gauge,
    meta: 'Action log',
  },
];

export function AutomationHub() {
  return (
    <PageChrome
      description="Catalogs, rules, and alert automation in one workspace."
      title="Automation and alerts"
    >
      <AutomationNav />

      <BentoSection title="Catalog">
        <HubLinkGrid>
          {CATALOG_LINKS.map((item) => (
            <HubLinkCard key={item.path} {...item} />
          ))}
        </HubLinkGrid>
      </BentoSection>

      <BentoSection title="Rules">
        <HubLinkGrid>
          {RULES_LINKS.map((item) => (
            <HubLinkCard key={item.path} {...item} />
          ))}
        </HubLinkGrid>
      </BentoSection>

      <BentoSection title="Monitoring">
        <HubLinkGrid>
          {MONITORING_LINKS.map((item) => (
            <HubLinkCard key={item.path} {...item} />
          ))}
        </HubLinkGrid>
      </BentoSection>
    </PageChrome>
  );
}
