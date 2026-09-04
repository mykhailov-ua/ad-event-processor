import {
  Activity,
  AppWindow,
  BarChart3,
  Crosshair,
  FileText,
  Globe,
  LayoutDashboard,
  Plug,
  ScrollText,
  Settings,
  Shield,
  Tag,
  Users,
  Wrench,
  type LucideIcon,
} from 'lucide-react';

import { filterNavItems, NAV_GROUPS, type NavItem } from '@/lib/nav_config';

export type TrackerNavItem = NavItem & {
  icon: LucideIcon;
};

const TRACKER_NAV_ICONS: Record<string, LucideIcon> = {
  '/dashboards/buyer': LayoutDashboard,
  '/customers': Users,
  '/campaigns': Crosshair,
  '/billing': BarChart3,
  '/team': Users,
  '/ops': Wrench,
  '/audit': ScrollText,
  '/reports': BarChart3,
  '/rtb': Activity,
  '/fraud': Shield,
  '/integrations': Plug,
  '/creative': FileText,
  '/automation': Activity,
  '/portals': AppWindow,
  '/settings': Settings,
  '/landers': FileText,
  '/offers': Tag,
  '/domains': Globe,
  '/flows': Activity,
  '/docs': ScrollText,
};

function iconForPath(path: string): LucideIcon {
  return TRACKER_NAV_ICONS[path] ?? Activity;
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
  return Activity;
}

/** Flat sidebar list in tracker-style order (dashboard and campaigns first). */
const TRACKER_NAV_ORDER: string[] = [
  '/dashboards/buyer',
  '/campaigns',
  '/landers',
  '/offers',
  '/integrations',
  '/reports',
  '/domains',
  '/customers',
  '/settings',
  '/creative',
  '/automation',
  '/billing',
  '/team',
  '/ops',
  '/audit',
  '/rtb',
  '/fraud',
  '/portals',
  '/docs',
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

export function listTrackerNavItems(permissions: string[] | undefined): TrackerNavItem[] {
  const flat = [...NAV_GROUPS.flatMap((group) => group.items), ...EXTRA_TRACKER_NAV];
  const filtered = filterNavItems(flat, permissions);
  const byPath = new Map(filtered.map((item) => [item.path, item]));

  const ordered: TrackerNavItem[] = [];
  const seen = new Set<string>();

  for (const path of TRACKER_NAV_ORDER) {
    const item = byPath.get(path);
    if (!item || seen.has(path)) {
      continue;
    }
    seen.add(path);
    ordered.push({
      ...item,
      label: TRACKER_NAV_LABELS[path] ?? item.label,
      icon: iconForPath(path),
    });
  }

  for (const item of filtered) {
    if (seen.has(item.path)) {
      continue;
    }
    ordered.push({
      ...item,
      label: TRACKER_NAV_LABELS[item.path] ?? item.label,
      icon: iconForPath(item.path),
    });
  }

  return ordered;
}
