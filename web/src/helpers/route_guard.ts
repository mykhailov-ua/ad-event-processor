import { can } from './permissions.js';

export type RouteAccess = {
  perms?: string[];
  altPerm?: string;
  roles?: string[];
};

const ROUTE_ACCESS: Record<string, RouteAccess> = {
  '/': {},
  '/bootstrap': {},
  '/install/done': {},
  '/customers': { perms: ['customers:read'] },
  '/customers/:id': { perms: ['customers:read'] },
  '/campaigns': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/campaigns/portfolio': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/campaigns/flows': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/campaigns/:id': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/campaigns/:id/telegram': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/billing': { perms: ['customers:read', 'billing:read', 'campaigns:read:masked'] },
  '/billing/invoices/:id': { perms: ['customers:read', 'billing:read', 'campaigns:read:masked'] },
  '/team': { perms: ['campaigns:read', 'billing:read'] },
  '/reports/placements': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/reports/keywords': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/reports': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/reports/:reportKey': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/ops': { perms: ['shards:read'] },
  '/ops/dlq': { perms: ['shards:read'] },
  '/ops/domains': { perms: ['settings:read'] },
  '/ops/shards': { perms: ['shards:read'] },
  '/ops/blacklist': { perms: ['shards:read'] },
  '/ops/recon': { perms: ['audit:read'] },
  '/ops/consent': { perms: ['shards:read'] },
  '/support/feedback': {},
  '/settings': { perms: ['settings:read'] },
  '/settings/license': { perms: ['customers:read'] },
  '/settings/domains': { perms: ['settings:read'] },
  '/audit': { perms: ['audit:read'], altPerm: 'settings:read' },
  '/rtb/deals': { perms: ['rtb:read'], altPerm: 'campaigns:read:masked' },
  '/rtb/integration': { perms: ['rtb:read'], altPerm: 'campaigns:read:masked' },
  '/margin-guard': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/integrations/margin-guard': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/smart-alerts': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/integrations/smart-alerts': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/integrations/schemas': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/publisher': { perms: ['supply:read:scoped'] },
  '/selfserve': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/selfserve/billing': { perms: ['customers:read'] },
  '/selfserve/api-keys': { perms: ['campaigns:write'] },
  '/selfserve/campaigns/new': { perms: ['campaigns:write'] },
  '/dev/components': { roles: ['A'] },
};

/**
 * Match a route pattern against a pathname and extract params.
 */
function matchPattern(pattern: string, pathname: string): Record<string, string> | null {
  const patternParts = pattern.split('/');
  const pathParts = pathname.split('/');
  if (patternParts.length !== pathParts.length) return null;
  const params: Record<string, string> = {};
  for (let i = 0; i < patternParts.length; i += 1) {
    const part = patternParts[i];
    const seg = pathParts[i];
    if (part.startsWith(':')) params[part.slice(1)] = seg;
    else if (part !== seg) return null;
  }
  return params;
}

/**
 * Resolve route access rules for a pathname.
 */
export function routeAccessFor(pathname: string): RouteAccess | null {
  const path = pathname.split('?')[0].replace(/\/$/, '') || '/';
  if (ROUTE_ACCESS[path]) return ROUTE_ACCESS[path];
  for (const pattern of Object.keys(ROUTE_ACCESS)) {
    if (pattern.includes(':') && matchPattern(pattern, path)) {
      return ROUTE_ACCESS[pattern];
    }
  }
  return null;
}

/**
 * Test whether the user may open the given admin route.
 */
export function canAccessRoute(pathname: string, permissions: string[], role = ''): boolean {
  const rules = routeAccessFor(pathname);
  if (!rules) return true;
  if (rules.roles?.length && !rules.roles.includes(role)) return false;
  if (!rules.perms?.length) return true;
  if (rules.perms.some((p) => can(permissions, p))) return true;
  if (rules.altPerm && can(permissions, rules.altPerm)) return true;
  return false;
}
