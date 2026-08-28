export type OpsCardDef = {
  id: string;
  route: string;
  title: string;
  description: string;
  perm?: string;
  altPerm?: string;
  live: boolean;
};

export const OPS_CARDS: OpsCardDef[] = [
  {
    id: 'shards',
    route: '/ops/shards',
    title: 'Shards',
    description: 'Redis shard health, stream lag, and shard 0 catch-up.',
    perm: 'shards:read',
    live: true,
  },
  {
    id: 'dlq',
    route: '/ops/dlq',
    title: 'DLQ',
    description: 'Dead-letter queues across shards and inbox retries.',
    perm: 'shards:read',
    live: true,
  },
  {
    id: 'blacklist',
    route: '/ops/blacklist',
    title: 'Blacklist',
    description: 'Manual IP blocks and fraud list entries.',
    perm: 'blacklist:read',
    live: true,
  },
  {
    id: 'domains',
    route: '/ops/domains',
    title: 'Domains',
    description: 'Rotation pools, TLS allowlist, and host health.',
    perm: 'settings:read',
    live: true,
  },
  {
    id: 'recon',
    route: '/ops/recon',
    title: 'Reconciliation',
    description: 'Budget and spend reconciliation run history.',
    perm: 'audit:read',
    live: true,
  },
  {
    id: 'fraud',
    route: '/fraud/decisions',
    title: 'Fraud admin',
    description: 'Decision explain lookup, ML labels, and integration health.',
    perm: 'audit:read',
    live: true,
  },
  {
    id: 'consent',
    route: '/ops/consent',
    title: 'Consent proofs',
    description: 'Recorded consent proof audit trail.',
    perm: 'shards:read',
    live: true,
  },
  {
    id: 'ml_model',
    route: '/ops/ml-model',
    title: 'ML model',
    description: 'Model versions, eval metrics, and manual labels.',
    perm: 'shards:read',
    live: true,
  },
  {
    id: 'edge_parity',
    route: '/ops/edge-parity',
    title: 'Edge parity',
    description: 'Edge vs tracker request differential report.',
    perm: 'campaigns:read',
    altPerm: 'campaigns:read:masked',
    live: true,
  },
];

export function visibleOpsCards(permissions: string[]): OpsCardDef[] {
  return OPS_CARDS.filter((card) => {
    if (!card.perm && !card.altPerm) return true;
    if (card.perm && permissions.includes(card.perm)) return true;
    if (card.altPerm && permissions.includes(card.altPerm)) return true;
    return false;
  });
}
