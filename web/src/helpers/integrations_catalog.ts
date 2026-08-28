export type IntegrationCardDef = {
  id: string;
  route: string;
  title: string;
  description: string;
  perm?: string;
  altPerm?: string;
  live: boolean;
};

export const INTEGRATION_CARDS: IntegrationCardDef[] = [
  {
    id: 'cost_sync',
    route: '/integrations/cost-sync',
    title: 'Cost sync',
    description: 'Ad network credentials and spend import history.',
    perm: 'campaigns:read',
    altPerm: 'campaigns:read:masked',
    live: true,
  },
  {
    id: 'postbacks',
    route: '/integrations/postbacks',
    title: 'Postbacks',
    description: 'CAPI delivery status and dead-letter queue retries.',
    perm: 'campaigns:read',
    altPerm: 'campaigns:read:masked',
    live: true,
  },
  {
    id: 'schemas',
    route: '/integrations/schemas',
    title: 'Integration schemas',
    description: 'Author and apply integration schema documents.',
    perm: 'campaigns:read',
    altPerm: 'campaigns:read:masked',
    live: true,
  },
  {
    id: 'templates',
    route: '/integration/templates/import',
    title: 'Template import',
    description: 'Import bundled traffic and affiliate templates.',
    perm: 'campaigns:read',
    altPerm: 'campaigns:read:masked',
    live: true,
  },
  {
    id: 'supply',
    route: '/integrations/supply',
    title: 'Supply files',
    description: 'sellers.json and ads.txt CRUD with validation.',
    perm: 'settings:read',
    live: true,
  },
  {
    id: 'margin_guard',
    route: '/integrations/margin-guard',
    title: 'Margin guard',
    description: 'ROI policies, activity log, and placement overrides.',
    perm: 'campaigns:read',
    altPerm: 'campaigns:read:masked',
    live: true,
  },
  {
    id: 'smart_alerts',
    route: '/integrations/smart-alerts',
    title: 'Smart alerts',
    description: 'Webhook alert rules and fired event history.',
    perm: 'campaigns:read',
    altPerm: 'campaigns:read:masked',
    live: true,
  },
  {
    id: 'automation',
    route: '/integrations/automation',
    title: 'Automation',
    description: 'Metric rules with presets and dry-run evaluation.',
    perm: 'campaigns:read',
    altPerm: 'campaigns:read:masked',
    live: true,
  },
];

export function visibleIntegrationCards(permissions: string[]): IntegrationCardDef[] {
  return INTEGRATION_CARDS.filter((card) => {
    if (!card.perm && !card.altPerm) return true;
    if (card.perm && permissions.includes(card.perm)) return true;
    if (card.altPerm && permissions.includes(card.altPerm)) return true;
    return false;
  });
}
