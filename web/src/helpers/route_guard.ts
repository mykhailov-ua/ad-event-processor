import { can, canAccessPublisher, canAccessSelfServe } from './permissions.js';
import * as auth from './auth.js';

export type RouteAccess = {
  perms?: string[];
  altPerm?: string;
  selfServe?: boolean;
  publisher?: boolean;
};

const ROUTE_ACCESS: Record<string, RouteAccess> = {
  '/': {},
  '/customers': { perms: ['customers:read'] },
  '/customers/:id': { perms: ['customers:read'] },
  '/campaigns': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/campaigns/:id': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/campaigns/:id/telegram': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/campaigns/wizard': { perms: ['campaigns:write'] },
  '/campaigns/landers/:id/editor': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/campaigns/flows': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/campaigns/flows/:id/builder': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/billing': { perms: ['customers:read'] },
  '/billing/invoices/:id': { perms: ['customers:read'] },
  '/rtb/deals': { perms: ['rtb:read'] },
  '/rtb/integration': { perms: ['rtb:read'] },
  '/brands': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/integrations': {},
  '/integrations/cost-sync': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/integrations/postbacks': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/integrations/schemas': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/integration/templates/import': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/integrations/supply': { perms: ['settings:read'] },
  '/integrations/margin-guard': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/integrations/smart-alerts': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/integrations/automation': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/margin-guard': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/smart-alerts': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/settings': { perms: ['settings:read'] },
  '/settings/license': { perms: ['customers:read'] },
  '/settings/domains': { perms: ['settings:read'] },
  '/settings/disputes': { perms: ['customers:read'] },
  '/settings/report-schedules': {
    perms: ['campaigns:read'],
    altPerm: 'campaigns:read:masked',
  },
  '/team': { perms: ['campaigns:read'], altPerm: 'billing:read' },
  '/support/feedback': {},
  '/audit': { perms: ['audit:read'] },
  '/fraud': { perms: ['audit:read'] },
  '/fraud/decisions': { perms: ['audit:read'] },
  '/fraud/labels': { perms: ['audit:read'] },
  '/fraud/overrides': { perms: ['audit:read'] },
  '/fraud/presets': { perms: ['audit:read'] },
  '/fraud/integrations': { perms: ['audit:read'] },
  '/ops': { perms: ['shards:read'] },
  '/ops/shards': { perms: ['shards:read'] },
  '/ops/dlq': { perms: ['shards:read'] },
  '/ops/domains': { perms: ['settings:read'] },
  '/ops/blacklist': { perms: ['blacklist:read'] },
  '/ops/recon': { perms: ['audit:read'] },
  '/ops/consent': { perms: ['shards:read'] },
  '/ops/ml-model': { perms: ['shards:read'] },
  '/ops/edge-parity': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/reports': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/reports/:reportKey': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked' },
  '/selfserve': { perms: ['campaigns:read'], altPerm: 'campaigns:read:masked', selfServe: true },
  '/selfserve/billing': { perms: ['customers:read'], selfServe: true },
  '/selfserve/api-keys': { perms: ['campaigns:write'], selfServe: true },
  '/selfserve/campaigns/new': { perms: ['campaigns:write'], selfServe: true },
  '/publisher': { publisher: true },
};

function matchPattern(pattern: string, pathname: string): boolean {
  const patternParts = pattern.split('/');
  const pathParts = pathname.split('/');
  if (patternParts.length !== pathParts.length) return false;
  for (let i = 0; i < patternParts.length; i += 1) {
    const part = patternParts[i];
    const seg = pathParts[i];
    if (part.startsWith(':')) continue;
    if (part !== seg) return false;
  }
  return true;
}

function routeAccessFor(pathname: string): RouteAccess | null {
  const path = pathname.split('?')[0].replace(/\/$/, '') || '/';
  if (ROUTE_ACCESS[path]) return ROUTE_ACCESS[path];
  for (const pattern of Object.keys(ROUTE_ACCESS)) {
    if (pattern.includes(':') && matchPattern(pattern, path)) {
      return ROUTE_ACCESS[pattern];
    }
  }
  return null;
}

export function canAccessRoute(pathname: string, permissions: string[]): boolean {
  const path = pathname.split('?')[0].replace(/\/$/, '') || '/';
  const user = auth.getUser();
  const role = user?.role;

  if (path.startsWith('/selfserve')) {
    if (!canAccessSelfServe(role, permissions)) return false;
  }
  if (path === '/publisher' || path.startsWith('/publisher/')) {
    if (!canAccessPublisher(permissions)) return false;
  }

  if (path === '/reports' || path.startsWith('/reports/')) {
    const rules = ROUTE_ACCESS['/reports'];
    if (!rules?.perms?.length) return true;
    if (rules.perms.some((perm) => can(permissions, perm))) return true;
    if (rules.altPerm && can(permissions, rules.altPerm)) return true;
    return false;
  }
  const rules = routeAccessFor(path);
  if (!rules) return true;
  if (rules.selfServe && !canAccessSelfServe(role, permissions)) return false;
  if (rules.publisher && !canAccessPublisher(permissions)) return false;
  if (!rules.perms?.length) return true;
  if (rules.perms.some((perm) => can(permissions, perm))) return true;
  if (rules.altPerm && can(permissions, rules.altPerm)) return true;
  return false;
}
