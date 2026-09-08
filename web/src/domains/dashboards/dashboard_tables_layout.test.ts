import assert from 'node:assert/strict';
import test from 'node:test';

import { buildDashboardTablesLayout } from '@/domains/dashboards/dashboard_tables_layout';

const section = (id: 'campaigns' | 'landers' | 'offers' | 'sources', title: string) => ({
  id,
  title,
  table: undefined,
});

test('buildDashboardTablesLayout places campaigns on top and pairs the rest', () => {
  const layout = buildDashboardTablesLayout(
    [
      section('campaigns', 'Top campaigns'),
      section('landers', 'Landing pages'),
      section('offers', 'Offers'),
      section('sources', 'Sources'),
    ],
    true
  );

  assert.equal(layout.top?.id, 'campaigns');
  assert.deepEqual(
    layout.pairRows.map((row) =>
      row.map((slot) => (slot.kind === 'breakdown' ? slot.section.id : slot.kind))
    ),
    [
      ['landers', 'offers'],
      ['sources', 'recent_clicks'],
    ]
  );
});

test('buildDashboardTablesLayout keeps a lone recent clicks row', () => {
  const layout = buildDashboardTablesLayout([section('campaigns', 'Campaigns')], true);

  assert.equal(layout.top?.id, 'campaigns');
  assert.deepEqual(layout.pairRows, [[{ kind: 'recent_clicks' }]]);
});
