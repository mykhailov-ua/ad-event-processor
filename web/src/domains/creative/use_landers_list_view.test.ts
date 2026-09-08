import assert from 'node:assert/strict';
import test from 'node:test';

import type { Lander } from '@/api/types';
import { countLandersByHosting, filterLanders } from '@/domains/creative/use_landers_list_view';

const rows: Lander[] = [
  {
    id: '00000000-0000-4000-8000-000000000001',
    name: 'External LP',
    url: 'https://example.com/a',
    created_at: '2026-01-01T00:00:00.000Z',
  },
  {
    id: '00000000-0000-4000-8000-000000000002',
    name: 'Hosted LP',
    hosted_asset_id: '00000000-0000-4000-8000-000000000099',
    hosted_url: 'https://trk.example/lp/2/',
    created_at: '2026-01-02T00:00:00.000Z',
  },
];

test('filterLanders applies hosting and search filters', () => {
  assert.equal(filterLanders(rows, '', 'external').length, 1);
  assert.equal(filterLanders(rows, '', 'hosted').length, 1);
  assert.equal(filterLanders(rows, 'hosted', '').length, 1);
});

test('countLandersByHosting tallies hosting kinds', () => {
  const counts = countLandersByHosting(rows);
  assert.equal(counts.total, 2);
  assert.equal(counts.external, 1);
  assert.equal(counts.hosted, 1);
});
