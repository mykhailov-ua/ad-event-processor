import { sidebarReportsNavLinks } from './nav_reports.js';

export type NavLink = {
  to: string;
  label: string;
  icon?: string;
  perm?: string;
  altPerm?: string;
};

export type NavGroup = {
  title: string;
  links: NavLink[];
};

export const NAV_OVERFLOW_LINKS: NavLink[] = [
  {
    to: '/dashboards/cfo',
    label: 'CFO dashboard',
    icon: 'file-spreadsheet',
    perm: 'customers:read',
  },
  {
    to: '/dashboards/accountant',
    label: 'Accountant dashboard',
    icon: 'receipt',
    perm: 'customers:read',
  },
  {
    to: '/dashboards/fraud',
    label: 'Fraud dashboard',
    icon: 'alert-triangle',
    perm: 'audit:read',
  },
  { to: '/ops/shards', label: 'Shards', icon: 'server', perm: 'shards:read' },
  { to: '/ops/ml-model', label: 'ML model', icon: 'bar-chart', perm: 'shards:read' },
  { to: '/ops/edge-parity', label: 'Edge parity', icon: 'activity', perm: 'shards:read' },
  { to: '/ops/dlq', label: 'DLQ inbox', icon: 'package', perm: 'shards:read' },
  { to: '/ops/domains', label: 'Domain rotation', icon: 'globe', perm: 'settings:read' },
  { to: '/ops/recon', label: 'Reconciliation', icon: 'git-compare', perm: 'audit:read' },
  { to: '/ops/blacklist', label: 'Blacklist', icon: 'shield-ban', perm: 'blacklist:read' },
  { to: '/ops/consent', label: 'Consent proof', icon: 'shield', perm: 'shards:read' },
  {
    to: '/integrations/cost-sync',
    label: 'Cost Sync',
    icon: 'refresh-cw',
    perm: 'campaigns:read',
    altPerm: 'campaigns:read:masked',
  },
  {
    to: '/margin-guard',
    label: 'Margin Guard',
    icon: 'trending-down',
    perm: 'campaigns:read',
    altPerm: 'campaigns:read:masked',
  },
  {
    to: '/integrations/smart-alerts',
    label: 'Smart Alerts',
    icon: 'bell',
    perm: 'campaigns:read',
    altPerm: 'campaigns:read:masked',
  },
  {
    to: '/integration/templates/import',
    label: 'Integration templates',
    icon: 'file-text',
    perm: 'campaigns:read',
    altPerm: 'campaigns:read:masked',
  },
  {
    to: '/integrations/supply',
    label: 'Supply files',
    icon: 'package',
    perm: 'settings:read',
  },
  {
    to: '/integrations/schemas',
    label: 'Integration schemas',
    icon: 'plug',
    perm: 'campaigns:read',
    altPerm: 'campaigns:read:masked',
  },
  { to: '/support/feedback', label: 'Support feedback', icon: 'send' },
];

export const NAV_GROUPS: NavGroup[] = [
  {
    title: 'Overview',
    links: [{ to: '/', label: 'Overview', icon: 'layout-dashboard' }],
  },
  {
    title: 'Campaigns',
    links: [
      {
        to: '/campaigns',
        label: 'Campaigns',
        icon: 'megaphone',
        perm: 'campaigns:read',
        altPerm: 'campaigns:read:masked',
      },
      {
        to: '/campaigns/portfolio',
        label: 'Portfolio',
        icon: 'bookmark',
        perm: 'campaigns:read',
        altPerm: 'campaigns:read:masked',
      },
      {
        to: '/campaigns/flows',
        label: 'Flows',
        icon: 'git-branch',
        perm: 'campaigns:read',
        altPerm: 'campaigns:read:masked',
      },
    ],
  },
  {
    title: 'Customers & billing',
    links: [
      { to: '/customers', label: 'Customers', icon: 'users', perm: 'customers:read' },
      {
        to: '/billing',
        label: 'Billing',
        icon: 'credit-card',
        perm: 'customers:read',
        altPerm: 'billing:read',
      },
      {
        to: '/team',
        label: 'Team',
        icon: 'users',
        perm: 'campaigns:read',
        altPerm: 'billing:read',
      },
    ],
  },
  {
    title: 'Analytics',
    links: [
      {
        to: '/dashboards/adops',
        label: 'Dashboards',
        icon: 'zap',
        perm: 'campaigns:read',
        altPerm: 'campaigns:read:masked',
      },
      ...sidebarReportsNavLinks(),
    ],
  },
  {
    title: 'RTB',
    links: [
      {
        to: '/rtb/integration',
        label: 'RTB integration',
        icon: 'radio',
        perm: 'rtb:read',
        altPerm: 'campaigns:read:masked',
      },
      {
        to: '/rtb/deals',
        label: 'PMP deals',
        icon: 'handshake',
        perm: 'rtb:read',
        altPerm: 'campaigns:read:masked',
      },
    ],
  },
  {
    title: 'Operations',
    links: [{ to: '/ops', label: 'Operations', icon: 'activity', perm: 'shards:read' }],
  },
  {
    title: 'Integrations',
    links: [
      {
        to: '/integrations/postbacks',
        label: 'Integrations',
        icon: 'plug',
        perm: 'campaigns:read',
        altPerm: 'campaigns:read:masked',
      },
    ],
  },
  {
    title: 'Admin',
    links: [
      { to: '/audit', label: 'Audit log', icon: 'scroll-text', perm: 'audit:read' },
      { to: '/settings', label: 'Platform', icon: 'settings', perm: 'settings:read' },
      { to: '/settings/license', label: 'License', icon: 'key', perm: 'customers:read' },
      { to: '/settings/domains', label: 'Domains', icon: 'globe', perm: 'settings:read' },
    ],
  },
  {
    title: 'Publisher',
    links: [
      {
        to: '/publisher',
        label: 'Publisher dashboard',
        icon: 'bar-chart-2',
        perm: 'supply:read:scoped',
      },
    ],
  },
];

export function navLinkVisible(permissions: string[], link: NavLink): boolean {
  if (!link.perm) return true;
  if (permissions.includes(link.perm)) return true;
  if (link.altPerm && permissions.includes(link.altPerm)) return true;
  return false;
}

export function visibleNavGroups(permissions: string[], role = ''): NavGroup[] {
  const pubOnly =
    role === 'P' ||
    (permissions.includes('supply:read:scoped') &&
      !permissions.includes('campaigns:read') &&
      !permissions.includes('campaigns:read:masked'));
  if (pubOnly) {
    return [
      {
        title: 'Publisher',
        links: [
          {
            to: '/publisher',
            label: 'Publisher dashboard',
            icon: 'bar-chart-2',
            perm: 'supply:read:scoped',
          },
        ],
      },
    ];
  }
  return NAV_GROUPS.map((group) => ({
    ...group,
    links: group.links.filter((l) => navLinkVisible(permissions, l)),
  })).filter((group) => group.links.length > 0);
}
