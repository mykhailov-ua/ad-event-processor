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
    title: 'Dashboards',
    links: [
      {
        to: '/dashboards/adops',
        label: 'AdOps',
        icon: 'zap',
        perm: 'campaigns:read',
        altPerm: 'campaigns:read:masked',
      },
      {
        to: '/dashboards/cfo',
        label: 'CFO',
        icon: 'file-spreadsheet',
        perm: 'customers:read',
      },
      {
        to: '/dashboards/accountant',
        label: 'Accountant',
        icon: 'receipt',
        perm: 'customers:read',
      },
      {
        to: '/dashboards/fraud',
        label: 'Fraud',
        icon: 'alert-triangle',
        perm: 'audit:read',
      },
    ],
  },
  {
    title: 'Reports',
    links: sidebarReportsNavLinks(),
  },
  {
    title: 'Deals',
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
    links: [
      { to: '/ops', label: 'Operations', icon: 'activity', perm: 'shards:read' },
      { to: '/ops/recon', label: 'Reconciliation', icon: 'git-compare', perm: 'audit:read' },
      { to: '/ops/blacklist', label: 'Blacklist', icon: 'shield-ban', perm: 'blacklist:read' },
    ],
  },
  {
    title: 'Security',
    links: [{ to: '/audit', label: 'Audit log', icon: 'scroll-text', perm: 'audit:read' }],
  },
  {
    title: 'Integrations',
    links: [
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
        to: '/integrations/supply',
        label: 'Supply files',
        icon: 'package',
        perm: 'settings:read',
      },
    ],
  },
  {
    title: 'Settings',
    links: [
      { to: '/settings', label: 'Platform', icon: 'settings', perm: 'settings:read' },
      { to: '/settings/license', label: 'License', icon: 'key', perm: 'customers:read' },
      { to: '/settings/domains', label: 'Domains', icon: 'globe', perm: 'settings:read' },
    ],
  },
];

/**
 * Test whether a nav link is visible for the given permissions.
 */
export function navLinkVisible(permissions: string[], link: NavLink): boolean {
  if (!link.perm) return true;
  if (permissions.includes(link.perm)) return true;
  if (link.altPerm && permissions.includes(link.altPerm)) return true;
  return false;
}

/**
 * Filter nav groups to links visible for the given permissions.
 */
export function visibleNavGroups(permissions: string[]): NavGroup[] {
  return NAV_GROUPS
    .map((group) => ({
      ...group,
      links: group.links.filter((l) => navLinkVisible(permissions, l)),
    }))
    .filter((group) => group.links.length > 0);
}
