import {
  Building2,
  Download,
  Megaphone,
  Plug,
  ScrollText,
  Settings,
  Users,
  Wrench,
  type LucideIcon,
} from 'lucide-react';

import {
  CONTROL_PLANE_NAV_ENABLED,
  filterControlPlaneNavGroups,
} from '@/lib/control_plane_scope';
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
  '/customers': Building2,
  '/campaigns': Megaphone,
  '/team': Users,
  '/settings': Settings,
  '/exports': Download,
  '/ops': Wrench,
  '/audit': ScrollText,
  '/integrations': Plug,
};

function iconForPath(path: string): LucideIcon {
  return TRACKER_NAV_ICONS[path] ?? Megaphone;
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
  return Megaphone;
}

function withIcons(items: NavItem[]): TrackerNavItem[] {
  return items.map((item) => ({
    ...item,
    icon: iconForPath(item.path),
  }));
}

export function listTrackerNavGroups(permissions: string[] | undefined): TrackerNavGroup[] {
  if (CONTROL_PLANE_NAV_ENABLED) {
    return filterControlPlaneNavGroups(permissions).map((group) => ({
      ...group,
      items: withIcons(group.items),
    }));
  }

  const flat = filterNavItems(NAV_GROUPS.flatMap((group) => group.items), permissions);
  const byPath = new Map<string, TrackerNavItem>();
  for (const item of flat) {
    if (!byPath.has(item.path)) {
      byPath.set(item.path, { ...item, icon: iconForPath(item.path) });
    }
  }
  return NAV_GROUPS.map((group) => ({
    ...group,
    items: group.items
      .map((item) => byPath.get(item.path))
      .filter((item): item is TrackerNavItem => item != null),
  })).filter((group) => group.items.length > 0);
}

export function listTrackerNavItems(permissions: string[] | undefined): TrackerNavItem[] {
  return listTrackerNavGroups(permissions).flatMap((group) => group.items);
}
