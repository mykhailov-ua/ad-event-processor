import { can } from './permissions.js';

/** @typedef {{ perms?: string[], altPerm?: string, roles?: string[] }} RouteAccess */

/** @type {Record<string, RouteAccess>} */
const ROUTE_ACCESS = {
  '/': {},
  '/bootstrap': {},
  '/customers': { perms: ['customers:read'] },
  '/customers/:id': { perms: ['customers:read'] },
  '/campaigns': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/campaigns/portfolio': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/campaigns/:id': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/billing': { perms: ['customers:read'], altPerm: 'billing:read' },
  '/billing/invoices/:id': { perms: ['customers:read'], altPerm: 'billing:read' },
  '/reports/placements': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/reports/keywords': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/reports': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/reports/:reportKey': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/ops': { perms: ['shards:read'] },
  '/ops/shards': { perms: ['shards:read'] },
  '/ops/blacklist': { perms: ['shards:read'] },
  '/settings': { perms: ['settings:read'] },
  '/audit': { perms: ['settings:read'] },
  '/rtb/deals': { perms: ['rtb:read'], altPerm: 'campaigns:read:masked' },
  '/dev/components': { roles: ['A'] },
};

/**
 * Match a route pattern against a pathname and extract params.
 *
 * @param {string} pattern
 * @param {string} pathname
 * @returns {Record<string, string>|null}
 */
function matchPattern(pattern, pathname) {
  const patternParts = pattern.split('/');
  const pathParts = pathname.split('/');
  if (patternParts.length !== pathParts.length) return null;
  /** @type {Record<string, string>} */
  const params = {};
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
 *
 * @param {string} pathname
 * @returns {RouteAccess|null}
 */
export function routeAccessFor(pathname) {
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
 *
 * @param {string} pathname
 * @param {string[]} permissions
 * @param {string} [role]
 * @returns {boolean}
 */
export function canAccessRoute(pathname, permissions, role = '') {
  const rules = routeAccessFor(pathname);
  if (!rules) return true;
  if (rules.roles?.length && !rules.roles.includes(role)) return false;
  if (!rules.perms?.length) return true;
  if (rules.perms.some((p) => can(permissions, p))) return true;
  if (rules.altPerm && can(permissions, rules.altPerm)) return true;
  return false;
}
