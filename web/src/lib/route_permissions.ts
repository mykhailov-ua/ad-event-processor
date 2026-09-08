import { NAV_GROUPS, type NavItem } from '@/lib/nav_config';

export type RoutePermission = Pick<NavItem, 'permission' | 'permissionAny'>;

type RoutePermissionRule = {
  prefix: string;
  permission?: string;
  permissionAny?: string[];
};

const EXTRA_ROUTE_RULES: RoutePermissionRule[] = [
  { prefix: '/landers', permission: 'campaigns:read' },
  { prefix: '/offers', permission: 'campaigns:read' },
  { prefix: '/domains', permission: 'campaigns:read' },
  { prefix: '/flows', permission: 'campaigns:read' },
  { prefix: '/brands', permission: 'campaigns:read' },
  { prefix: '/brand-creatives', permission: 'campaigns:read' },
  { prefix: '/supply', permission: 'supply:read:scoped' },
  {
    prefix: '/dashboards',
    permissionAny: ['campaigns:read', 'campaigns:read:masked'],
  },
  { prefix: '/traffic-optimizer', permission: 'campaigns:read' },
  { prefix: '/smart-alerts', permission: 'campaigns:read' },
  { prefix: '/margin-guard', permission: 'campaigns:read' },
  { prefix: '/telegram', permission: 'campaigns:read' },
  { prefix: '/selfserve', permission: 'campaigns:read' },
  { prefix: '/publisher', permission: 'supply:read:scoped' },
  { prefix: '/report-schedules', permission: 'campaigns:read' },
  { prefix: '/views', permission: 'campaigns:read' },
  { prefix: '/forecast', permission: 'campaigns:read' },
  { prefix: '/disputes', permission: 'customers:read' },
  { prefix: '/support', permission: 'campaigns:read' },
  { prefix: '/docs', permissionAny: ['shards:read', 'audit:read', 'campaigns:read'] },
];

function buildRouteRules(): RoutePermissionRule[] {
  const fromNav = NAV_GROUPS.flatMap((group) => group.items).map((item) => ({
    prefix: item.path,
    permission: item.permission,
    permissionAny: item.permissionAny,
  }));
  return [...fromNav, ...EXTRA_ROUTE_RULES].sort((left, right) => right.prefix.length - left.prefix.length);
}

const ROUTE_RULES = buildRouteRules();

export function resolveRoutePermission(pathname: string): RoutePermission | null {
  const normalized = pathname.replace(/\/$/, '') || '/';
  for (const rule of ROUTE_RULES) {
    if (normalized === rule.prefix || normalized.startsWith(`${rule.prefix}/`)) {
      return rule;
    }
  }
  return null;
}

export function sessionHasRoutePermission(
  permissions: string[] | undefined,
  rule: RoutePermission | null
): boolean {
  if (!rule) {
    return true;
  }
  if (permissions === undefined) {
    return true;
  }
  if (rule.permissionAny) {
    return rule.permissionAny.some((permission) => permissions.includes(permission));
  }
  if (rule.permission) {
    return permissions.includes(rule.permission);
  }
  return true;
}
