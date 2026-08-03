/**
 * @typedef {Object} NavLink
 * @property {string} to
 * @property {string} label
 * @property {string} [icon]
 * @property {string} [perm]
 * @property {string} [altPerm]
 */

/**
 * @typedef {Object} NavGroup
 * @property {string} title
 * @property {NavLink[]} links
 */

/** @type {NavGroup[]} */
export const NAV_GROUPS = [
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
    ],
  },
  {
    title: 'Customers & billing',
    links: [
      { to: '/customers', label: 'Customers', icon: 'users', perm: 'customers:read' },
      { to: '/billing', label: 'Billing', icon: 'credit-card', perm: 'customers:read', altPerm: 'billing:read' },
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
        icon: 'file-text',
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
    links: [
      {
        to: '/reports',
        label: 'Reports hub',
        icon: 'bar-chart',
        perm: 'campaigns:read',
        altPerm: 'campaigns:read:masked',
      },
      {
        to: '/reports/placements',
        label: 'Placements',
        icon: 'globe',
        perm: 'campaigns:read',
        altPerm: 'campaigns:read:masked',
      },
      {
        to: '/reports/keywords',
        label: 'Keywords',
        icon: 'arrow-up-down',
        perm: 'campaigns:read',
        altPerm: 'campaigns:read:masked',
      },
      {
        to: '/reports/ivt-by-source',
        label: 'IVT by source',
        icon: 'alert-circle',
        perm: 'audit:read',
      },
    ],
  },
  {
    title: 'Deals',
    links: [
      {
        to: '/rtb/deals',
        label: 'Deal performance',
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
    title: 'Settings',
    links: [{ to: '/settings', label: 'Platform', icon: 'settings', perm: 'settings:read' }],
  },
];

/**
 * Test whether a nav link is visible for the given permissions.
 *
 * @param {string[]} permissions
 * @param {NavLink} link
 * @returns {boolean}
 */
export function navLinkVisible(permissions, link) {
  if (!link.perm) return true;
  if (permissions.includes(link.perm)) return true;
  if (link.altPerm && permissions.includes(link.altPerm)) return true;
  return false;
}

/**
 * Filter nav groups to links visible for the given permissions.
 *
 * @param {string[]} permissions
 * @returns {NavGroup[]}
 */
export function visibleNavGroups(permissions) {
  return NAV_GROUPS
    .map((group) => ({
      ...group,
      links: group.links.filter((l) => navLinkVisible(permissions, l)),
    }))
    .filter((group) => group.links.length > 0);
}
