import { sessionHasAnyPermission, sessionHasPermission } from '@/lib/session_permissions';

export type NavItem = {
  path: string;
  label: string;
  permission?: string;
  permissionAny?: string[];
  badgeCount?: number;
};

export type NavGroup = {
  id: string;
  label: string;
  items: NavItem[];
};

const CORE_NAV: NavItem[] = [
  { path: '/customers', label: 'Customers', permission: 'customers:read' },
  {
    path: '/campaigns',
    label: 'Campaigns',
    permissionAny: ['campaigns:read', 'campaigns:read:masked'],
  },
  { path: '/team', label: 'Team', permissionAny: ['team:read', 'campaigns:read'] },
  { path: '/settings', label: 'Settings', permission: 'settings:read' },
];

const OPERATIONS_NAV: NavItem[] = [
  {
    path: '/dashboards/buyer',
    label: 'Dashboard',
    permissionAny: ['campaigns:read', 'campaigns:read:masked'],
  },
  { path: '/exports', label: 'Exports', permission: 'campaigns:read' },
  { path: '/alerts', label: 'Alerts', permission: 'campaigns:read' },
  { path: '/ops', label: 'Ops', permission: 'shards:read' },
  { path: '/audit', label: 'Audit', permission: 'audit:read' },
  { path: '/integrations', label: 'Integrations', permission: 'campaigns:read' },
];

export const NAV_GROUPS: NavGroup[] = [
  { id: 'core', label: 'Core', items: CORE_NAV },
  { id: 'operations', label: 'Operations', items: OPERATIONS_NAV },
];

export const NAV_ITEMS: NavItem[] = NAV_GROUPS.flatMap((group) => group.items);

export function filterNavItems(items: NavItem[], permissions: string[] | undefined): NavItem[] {
  if (permissions === undefined) {
    return items;
  }
  return items.filter((item) => {
    if (item.permissionAny) {
      return sessionHasAnyPermission(permissions, item.permissionAny);
    }
    if (item.permission) {
      return sessionHasPermission(permissions, item.permission);
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
