import { can } from './permissions.js';

export type SelfServeCardDef = {
  id: string;
  route: string;
  title: string;
  description: string;
  perm?: string;
  altPerm?: string;
};

export const SELFSERVE_CARDS: SelfServeCardDef[] = [
  {
    id: 'portfolio',
    route: '/selfserve',
    title: 'Portfolio',
    description: 'Campaign performance and pause or resume controls.',
    perm: 'campaigns:read',
    altPerm: 'campaigns:read:masked',
  },
  {
    id: 'billing',
    route: '/selfserve/billing',
    title: 'Billing',
    description: 'Statement lines and invoice history for your account.',
    perm: 'customers:read',
  },
  {
    id: 'api_keys',
    route: '/selfserve/api-keys',
    title: 'API keys',
    description: 'Create integration keys. List endpoint is not shipped.',
    perm: 'campaigns:write',
  },
  {
    id: 'create',
    route: '/selfserve/campaigns/new',
    title: 'New campaign',
    description: 'Quick create from a self-serve template.',
    perm: 'campaigns:write',
  },
];

export function visibleSelfServeCards(permissions: string[]): SelfServeCardDef[] {
  return SELFSERVE_CARDS.filter((card) => {
    if (!card.perm && !card.altPerm) return true;
    if (card.perm && can(permissions, card.perm)) return true;
    if (card.altPerm && can(permissions, card.altPerm)) return true;
    return false;
  });
}
