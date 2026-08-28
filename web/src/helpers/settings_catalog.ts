export type SettingsCardDef = {
  id: string;
  route: string;
  title: string;
  description: string;
  perm?: string;
  altPerm?: string;
  live: boolean;
};

export const SETTINGS_CARDS: SettingsCardDef[] = [
  {
    id: 'license',
    route: '/settings/license',
    title: 'License',
    description: 'JWT license status, HWID diagnostics, and token apply.',
    perm: 'customers:read',
    live: true,
  },
  {
    id: 'domains',
    route: '/settings/domains',
    title: 'Domains',
    description: 'Custom tracking domains, SSL setup, probes, and parking.',
    perm: 'settings:read',
    live: true,
  },
  {
    id: 'disputes',
    route: '/settings/disputes',
    title: 'Disputes',
    description: 'Payment chargeback and dispute ledger rows.',
    perm: 'customers:read',
    live: true,
  },
  {
    id: 'report_schedules',
    route: '/settings/report-schedules',
    title: 'Report schedules',
    description: 'Cron-driven report exports per customer.',
    perm: 'campaigns:read',
    altPerm: 'campaigns:read:masked',
    live: true,
  },
  {
    id: 'team',
    route: '/team',
    title: 'Team',
    description: 'Roster, invites, spend caps, and budget approvals.',
    perm: 'campaigns:read',
    altPerm: 'billing:read',
    live: true,
  },
  {
    id: 'feedback',
    route: '/support/feedback',
    title: 'Feedback',
    description: 'Operator bug reports and feature requests.',
    live: true,
  },
];

export function visibleSettingsCards(permissions: string[]): SettingsCardDef[] {
  return SETTINGS_CARDS.filter((card) => {
    if (!card.perm && !card.altPerm) return true;
    if (card.perm && permissions.includes(card.perm)) return true;
    if (card.altPerm && permissions.includes(card.altPerm)) return true;
    return false;
  });
}
