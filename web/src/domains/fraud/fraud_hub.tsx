import { Brain, FileSearch, Link2, Scale, ShieldAlert, BarChart3 } from 'lucide-react';

import { BentoSection } from '@/shell/bento_card';
import { HubLinkCard, HubLinkGrid } from '@/shell/hub_link_card';
import { PageChrome } from '@/shell/page_chrome';
import { FRAUD_HUB_REPORT_LINKS } from '@/domains/fraud/fraud_hub_links';

const FRAUD_LINKS = [
  {
    path: '/fraud/integrations',
    title: 'Integrations',
    description: 'Third-party fraud provider health and DLQ counts per campaign.',
    icon: Link2,
    meta: 'Provider health',
  },
  {
    path: '/fraud/labels',
    title: 'ML labels',
    description: 'Manual fraud labels for IP hashes used in model training.',
    icon: Brain,
    meta: 'Training labels',
  },
  {
    path: '/fraud/overrides',
    title: 'Scoring overrides',
    description: 'Per-campaign or per-IP fraud scoring overrides.',
    icon: Scale,
    meta: 'Score overrides',
  },
  {
    path: '/fraud/presets',
    title: 'Policy presets',
    description: 'Global fraud sensitivity preset thresholds.',
    icon: ShieldAlert,
    meta: 'Sensitivity tiers',
  },
  {
    path: '/fraud/moderator-corpus',
    title: 'Moderator corpus',
    description: 'Learned moderator fingerprint tuples for review-traffic safe-page routing.',
    icon: FileSearch,
    meta: 'Review routing',
  },
  {
    path: '/docs/fraud-signal-limits',
    title: 'Signal limits',
    description: 'Safe-page, residential proxy, and ML batch enforcement limits.',
    icon: ShieldAlert,
    meta: 'Operator doc',
  },
  {
    path: '/docs/perimeter-sybil-controls',
    title: 'Sybil runbook (T2)',
    description: 'Human operator limits, crowd-wave response, and honest perimeter copy.',
    icon: ShieldAlert,
    meta: 'Perimeter doc',
  },
  {
    path: '/fraud/decisions',
    title: 'Decision explain',
    description: 'Explain fraud tier decision for an IP hash.',
    icon: FileSearch,
    meta: 'Explainability',
  },
];

export function FraudHub() {
  return (
    <PageChrome
      description="Integrations, labels, overrides, analytics, and decision explain."
      title="Fraud"
    >
      <BentoSection title="Fraud operations">
        <HubLinkGrid>
          {FRAUD_LINKS.map((item) => (
            <HubLinkCard key={item.path} {...item} />
          ))}
        </HubLinkGrid>
      </BentoSection>
      <BentoSection title="Fraud analytics">
        <HubLinkGrid>
          {FRAUD_HUB_REPORT_LINKS.map((item) => (
            <HubLinkCard
              key={item.path}
              description={item.description}
              icon={BarChart3}
              meta={item.meta}
              path={item.path}
              title={item.title}
            />
          ))}
        </HubLinkGrid>
      </BentoSection>
    </PageChrome>
  );
}
