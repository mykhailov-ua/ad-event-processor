import { PORTAL_PERMISSIONS } from '@/lib/portal_access';

export type NavItem = {
  path: string;
  label: string;
  permission?: string;
  permissionAny?: string[];
};

export type NavGroup = {
  id: string;
  label: string;
  items: NavItem[];
};

const CORE_NAV: NavItem[] = [
  { path: '/customers', label: 'Customers', permission: 'customers:read' },
  { path: '/campaigns', label: 'Campaigns', permission: 'campaigns:read' },
  { path: '/billing', label: 'Billing', permission: 'customers:read' },
  {
    path: '/dashboards/buyer',
    label: 'Dashboards',
    permissionAny: ['campaigns:read', 'campaigns:read:masked'],
  },
  { path: '/team', label: 'Team', permission: 'campaigns:read' },
];

const OPERATIONS_NAV: NavItem[] = [
  { path: '/ops', label: 'Ops', permission: 'shards:read' },
  { path: '/audit', label: 'Audit', permission: 'audit:read' },
  { path: '/reports', label: 'Reports', permission: 'campaigns:read' },
];

const PLATFORM_NAV: NavItem[] = [
  { path: '/rtb', label: 'RTB', permission: 'rtb:read' },
  { path: '/fraud', label: 'Fraud', permission: 'audit:read' },
  { path: '/integrations', label: 'Integrations', permission: 'campaigns:read' },
];

const CONTENT_NAV: NavItem[] = [
  { path: '/creative', label: 'Creative', permission: 'campaigns:read' },
  { path: '/automation', label: 'Automation', permission: 'campaigns:read' },
];

const MORE_NAV: NavItem[] = [
  {
    path: '/portals',
    label: 'Portals',
    permissionAny: Object.values(PORTAL_PERMISSIONS),
  },
  { path: '/settings', label: 'Settings', permission: 'settings:read' },
];

export const NAV_GROUPS: NavGroup[] = [
  { id: 'core', label: 'Core', items: CORE_NAV },
  { id: 'operations', label: 'Operations', items: OPERATIONS_NAV },
  { id: 'platform', label: 'Platform', items: PLATFORM_NAV },
  { id: 'content', label: 'Content', items: CONTENT_NAV },
  { id: 'more', label: 'More', items: MORE_NAV },
];

/** Flat list for command palette and legacy callers. */
export const NAV_ITEMS: NavItem[] = NAV_GROUPS.flatMap((group) => group.items);

// UX-only nav visibility; server RBAC on /api/v1 is authoritative (RB-L2).
export function filterNavItems(items: NavItem[], permissions: string[] | undefined): NavItem[] {
  if (permissions === undefined) {
    return items;
  }
  return items.filter((item) => {
    if (item.permissionAny) {
      return item.permissionAny.some((permission) => permissions.includes(permission));
    }
    if (item.permission) {
      return permissions.includes(item.permission);
    }
    return true;
  });
}

export function filterNavGroups(groups: NavGroup[], permissions: string[] | undefined): NavGroup[] {
  return groups
    .map((group) => ({
      ...group,
      items: filterNavItems(group.items, permissions),
    }))
    .filter((group) => group.items.length > 0);
}

export type SectionNavItem = {
  path: string;
  label: string;
  exact?: boolean;
};

export function isSectionNavActive(pathname: string, item: SectionNavItem): boolean {
  if (item.exact) {
    return pathname === item.path;
  }
  return pathname === item.path || pathname.startsWith(`${item.path}/`);
}
