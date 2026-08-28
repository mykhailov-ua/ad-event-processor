import { canAccessPublisher, canAccessSelfServe } from './permissions.js';

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
  selfServeOnly?: boolean;
  publisherOnly?: boolean;
};

const NAV_GROUPS: NavGroup[] = [
  {
    title: 'Directory',
    links: [{ to: '/customers', label: 'Customers', icon: 'users', perm: 'customers:read' }],
  },
  {
    title: 'Commercial',
    links: [
      {
        to: '/campaigns',
        label: 'Campaigns',
        icon: 'megaphone',
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
      {
        to: '/campaigns/wizard',
        label: 'Campaign wizard',
        icon: 'wand',
        perm: 'campaigns:write',
      },
      { to: '/billing', label: 'Billing', icon: 'receipt', perm: 'customers:read' },
      {
        to: '/brands',
        label: 'Brands',
        icon: 'tag',
        perm: 'campaigns:read',
        altPerm: 'campaigns:read:masked',
      },
    ],
  },
  {
    title: 'Self-serve',
    selfServeOnly: true,
    links: [
      {
        to: '/selfserve',
        label: 'Portfolio',
        icon: 'layout-dashboard',
        perm: 'campaigns:read',
        altPerm: 'campaigns:read:masked',
      },
      {
        to: '/selfserve/billing',
        label: 'Billing',
        icon: 'wallet',
        perm: 'customers:read',
      },
      {
        to: '/selfserve/api-keys',
        label: 'API keys',
        icon: 'key',
        perm: 'campaigns:write',
      },
      {
        to: '/selfserve/campaigns/new',
        label: 'New campaign',
        icon: 'plus',
        perm: 'campaigns:write',
      },
    ],
  },
  {
    title: 'Publisher',
    publisherOnly: true,
    links: [
      {
        to: '/publisher',
        label: 'Publisher portal',
        icon: 'globe',
        perm: 'supply:read:scoped',
      },
    ],
  },
  {
    title: 'Integrations',
    links: [
      { to: '/integrations', label: 'Hub', icon: 'plug', perm: 'campaigns:read', altPerm: 'campaigns:read:masked' },
      { to: '/integrations/cost-sync', label: 'Cost sync', icon: 'refresh', perm: 'campaigns:read', altPerm: 'campaigns:read:masked' },
      { to: '/integrations/postbacks', label: 'Postbacks', icon: 'webhook', perm: 'campaigns:read', altPerm: 'campaigns:read:masked' },
      { to: '/integrations/supply', label: 'Supply', icon: 'file', perm: 'settings:read' },
    ],
  },
  {
    title: 'RTB',
    links: [
      { to: '/rtb/deals', label: 'Deals', icon: 'handshake', perm: 'rtb:read' },
      { to: '/rtb/integration', label: 'Integration', icon: 'settings', perm: 'rtb:read' },
    ],
  },
  {
    title: 'Reports',
    links: [
      {
        to: '/reports',
        label: 'Reports hub',
        icon: 'chart',
        perm: 'campaigns:read',
        altPerm: 'campaigns:read:masked',
      },
    ],
  },
  {
    title: 'Fraud',
    links: [
      { to: '/fraud/decisions', label: 'Decisions', icon: 'search', perm: 'audit:read' },
      { to: '/fraud/labels', label: 'Labels', icon: 'tag', perm: 'audit:read' },
      { to: '/fraud/overrides', label: 'Overrides', icon: 'shield-off', perm: 'audit:read' },
      { to: '/fraud/presets', label: 'Presets', icon: 'sliders', perm: 'audit:read' },
      { to: '/fraud/integrations', label: 'Integrations', icon: 'plug', perm: 'audit:read' },
    ],
  },
  {
    title: 'Ops',
    links: [
      { to: '/ops', label: 'Operations', icon: 'activity', perm: 'shards:read' },
      { to: '/ops/shards', label: 'Shards', icon: 'database', perm: 'shards:read' },
      { to: '/ops/dlq', label: 'DLQ', icon: 'inbox', perm: 'shards:read' },
      { to: '/ops/blacklist', label: 'Blacklist', icon: 'shield', perm: 'blacklist:read' },
    ],
  },
  {
    title: 'Settings',
    links: [
      { to: '/settings', label: 'Platform', icon: 'settings', perm: 'settings:read' },
      { to: '/settings/license', label: 'License', icon: 'key', perm: 'customers:read' },
      { to: '/settings/domains', label: 'Domains', icon: 'globe', perm: 'settings:read' },
      {
        to: '/team',
        label: 'Team',
        icon: 'users',
        perm: 'campaigns:read',
        altPerm: 'billing:read',
      },
      { to: '/support/feedback', label: 'Feedback', icon: 'message' },
      { to: '/settings/disputes', label: 'Disputes', icon: 'alert', perm: 'customers:read' },
      {
        to: '/settings/report-schedules',
        label: 'Report schedules',
        icon: 'calendar',
        perm: 'campaigns:read',
        altPerm: 'campaigns:read:masked',
      },
      { to: '/audit', label: 'Audit', icon: 'list', perm: 'audit:read' },
    ],
  },
];

function linkVisible(permissions: string[], link: NavLink): boolean {
  if (!link.perm && !link.altPerm) return true;
  if (link.perm && permissions.includes(link.perm)) return true;
  if (link.altPerm && permissions.includes(link.altPerm)) return true;
  return false;
}

export function visibleNavGroups(permissions: string[], role: string): NavGroup[] {
  return NAV_GROUPS.filter((group) => {
    if (group.selfServeOnly && !canAccessSelfServe(role, permissions)) return false;
    if (group.publisherOnly && !canAccessPublisher(permissions)) return false;
    return true;
  })
    .map((group) => ({
      ...group,
      links: group.links.filter((link) => linkVisible(permissions, link)),
    }))
    .filter((group) => group.links.length > 0);
}
