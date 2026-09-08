import {
  AppWindow,
  BarChart3,
  BookOpen,
  Building2,
  Gavel,
  GitBranch,
  Globe,
  LayoutDashboard,
  LayoutTemplate,
  Layers,
  Megaphone,
  Palette,
  Plug,
  Receipt,
  ScrollText,
  Settings,
  ShieldAlert,
  Tag,
  Users,
  Workflow,
  Wrench,
  type LucideIcon,
} from 'lucide-react';

import { filterNavItems, NAV_GROUPS, type NavItem } from '@/lib/nav_config';

export type TrackerNavItem = NavItem & {
  icon: LucideIcon;
};

export type TrackerNavGroup = {
  id: string;
  label: string;
  items: TrackerNavItem[];
};

const TRACKER_NAV_ICONS: Record<string, LucideIcon> = {
  '/dashboards/buyer': LayoutDashboard,
  '/customers': Building2,
  '/campaigns': Megaphone,
  '/billing': Receipt,
  '/team': Users,
  '/ops': Wrench,
  '/audit': ScrollText,
  '/reports': BarChart3,
  '/rtb': Gavel,
  '/fraud': ShieldAlert,
  '/integrations': Plug,
  '/creative': Palette,
  '/automation': Workflow,
  '/portals': AppWindow,
  '/settings': Settings,
  '/landers': LayoutTemplate,
  '/offers': Tag,
  '/domains': Globe,
  '/flows': GitBranch,
  '/docs': BookOpen,
};

function iconForPath(path: string): LucideIcon {
  return TRACKER_NAV_ICONS[path] ?? Layers;
}

/** Lucide icon for the current pathname (longest registered nav prefix). */
export function trackerNavIconForPathname(pathname: string): LucideIcon {
  const normalized = pathname.replace(/\/$/, '') || '/';
  if (TRACKER_NAV_ICONS[normalized]) {
    return TRACKER_NAV_ICONS[normalized];
  }
  const parts = normalized.split('/').filter(Boolean);
  for (let depth = parts.length; depth >= 1; depth -= 1) {
    const candidate = `/${parts.slice(0, depth).join('/')}`;
    const icon = TRACKER_NAV_ICONS[candidate];
    if (icon) {
      return icon;
    }
  }
  return Layers;
}

/** Sidebar sections in tracker-style order (paths within each group). */
const TRACKER_NAV_GROUP_DEFS: ReadonlyArray<{
  id: string;
  label: string;
  paths: readonly string[];
}> = [
  { id: 'overview', label: 'Overview', paths: ['/dashboards/buyer'] },
  {
    id: 'traffic',
    label: 'Traffic',
    paths: ['/campaigns', '/landers', '/offers', '/domains', '/creative'],
  },
  { id: 'analytics', label: 'Analytics', paths: ['/reports'] },
  {
    id: 'platform',
    label: 'Platform',
    paths: ['/integrations', '/automation', '/rtb', '/fraud', '/portals'],
  },
  {
    id: 'organization',
    label: 'Organization',
    paths: ['/customers', '/team', '/billing', '/settings'],
  },
  { id: 'operations', label: 'Operations', paths: ['/ops', '/audit', '/docs'] },
];

const TRACKER_NAV_LABELS: Record<string, string> = {
  '/dashboards/buyer': 'Dashboard',
  '/campaigns': 'Campaigns',
  '/landers': 'Landing Pages',
  '/offers': 'Offers',
  '/integrations': 'Integrations',
  '/reports': 'Reports',
  '/domains': 'Domains',
  '/customers': 'Users',
  '/settings': 'Settings',
  '/creative': 'Creative',
  '/automation': 'Automation',
  '/billing': 'Billing',
  '/team': 'Team',
  '/ops': 'Maintenance',
  '/audit': 'Audit',
  '/rtb': 'RTB',
  '/fraud': 'Fraud',
  '/portals': 'Portals',
  '/docs': 'Documentation',
};

const EXTRA_TRACKER_NAV: NavItem[] = [
  { path: '/landers', label: 'Landing Pages', permission: 'campaigns:read' },
  { path: '/offers', label: 'Offers', permission: 'campaigns:read' },
  { path: '/domains', label: 'Domains', permission: 'campaigns:read' },
];

function buildTrackerNavItemMap(
  permissions: string[] | undefined
): Map<string, TrackerNavItem> {
  const flat = [...NAV_GROUPS.flatMap((group) => group.items), ...EXTRA_TRACKER_NAV];
  const filtered = filterNavItems(flat, permissions);
  const byPath = new Map<string, TrackerNavItem>();

  for (const item of filtered) {
    if (byPath.has(item.path)) {
      continue;
    }
    byPath.set(item.path, {
      ...item,
      label: TRACKER_NAV_LABELS[item.path] ?? item.label,
      icon: iconForPath(item.path),
    });
  }

  return byPath;
}

export function listTrackerNavGroups(permissions: string[] | undefined): TrackerNavGroup[] {
  const byPath = buildTrackerNavItemMap(permissions);
  const groups: TrackerNavGroup[] = [];
  const assigned = new Set<string>();

  for (const def of TRACKER_NAV_GROUP_DEFS) {
    const items: TrackerNavItem[] = [];
    for (const path of def.paths) {
      const item = byPath.get(path);
      if (!item || assigned.has(path)) {
        continue;
      }
      assigned.add(path);
      items.push(item);
    }
    if (items.length > 0) {
      groups.push({ id: def.id, label: def.label, items });
    }
  }

  const remainder: TrackerNavItem[] = [];
  for (const [path, item] of byPath) {
    if (!assigned.has(path)) {
      remainder.push(item);
    }
  }
  if (remainder.length > 0) {
    groups.push({ id: 'more', label: 'More', items: remainder });
  }

  return groups;
}

export function listTrackerNavItems(permissions: string[] | undefined): TrackerNavItem[] {
  return listTrackerNavGroups(permissions).flatMap((group) => group.items);
}
