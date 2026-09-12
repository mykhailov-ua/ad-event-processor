import { NAV_GROUPS, type NavItem } from '@/lib/nav_config';
import { sessionHasAnyPermission, sessionHasPermission } from '@/lib/session_permissions';

export type RoutePermission = Pick<NavItem, 'permission' | 'permissionAny'>;

type RoutePermissionRule = {
  prefix: string;
  permission?: string;
  permissionAny?: string[];
};

const EXTRA_ROUTE_RULES: RoutePermissionRule[] = [
  { prefix: '/billing/invoices', permission: 'customers:read' },
  { prefix: '/reports/jobs', permission: 'campaigns:read' },
  { prefix: '/billing/exports', permission: 'campaigns:read' },
  { prefix: '/settings/access', permission: 'access:read' },
  {
    prefix: '/dashboards',
    permissionAny: ['campaigns:read', 'campaigns:read:masked'],
  },
  { prefix: '/exports/schedules', permission: 'campaigns:read' },
  { prefix: '/alerts', permission: 'campaigns:read' },
  { prefix: '/disputes', permission: 'customers:read' },
];

function buildRouteRules(): RoutePermissionRule[] {
  const fromNav = NAV_GROUPS.flatMap((group) => group.items).map((item) => ({
    prefix: item.path,
    permission: item.permission,
    permissionAny: item.permissionAny,
  }));
  return [...fromNav, ...EXTRA_ROUTE_RULES].sort(
    (left, right) => right.prefix.length - left.prefix.length
  );
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

export function formatRoutePermissionRequirement(rule: RoutePermission | null): string | undefined {
  if (!rule) {
    return undefined;
  }
  if (rule.permissionAny?.length) {
    return rule.permissionAny.join(' or ');
  }
  if (rule.permission) {
    return rule.permission;
  }
  return undefined;
}

export function sessionHasRoutePermission(
  permissions: string[] | undefined,
  rule: RoutePermission | null
): boolean {
  if (!rule) {
    return true;
  }
  if (permissions === undefined) {
    return false;
  }
  if (rule.permissionAny) {
    return sessionHasAnyPermission(permissions, rule.permissionAny);
  }
  if (rule.permission) {
    return sessionHasPermission(permissions, rule.permission);
  }
  return true;
}
