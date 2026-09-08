import assert from 'node:assert/strict';
import test from 'node:test';

import { buildBreadcrumbs } from './breadcrumbs.ts';

test('buildBreadcrumbs links campaign list but not bare campaign id before edit', () => {
  const campaignId = '550e8400-e29b-41d4-a716-446655440000';
  const crumbs = buildBreadcrumbs(`/campaigns/${campaignId}/edit`, {
    [campaignId]: 'Summer promo',
  });

  assert.deepEqual(
    crumbs.map((crumb) => crumb.label),
    ['Campaigns', 'Summer promo', 'Editor']
  );
  assert.equal(crumbs[0]?.href, '/campaigns');
  assert.equal(crumbs[1]?.href, undefined);
  assert.equal(crumbs[2]?.href, undefined);
});

test('buildBreadcrumbs does not link billing invoices index without a list route', () => {
  const invoiceId = '550e8400-e29b-41d4-a716-446655440001';
  const crumbs = buildBreadcrumbs(`/billing/invoices/${invoiceId}`, {
    [invoiceId]: 'INV-42',
  });

  assert.deepEqual(
    crumbs.map((crumb) => crumb.label),
    ['Billing', 'Invoices', 'INV-42']
  );
  assert.equal(crumbs[0]?.href, '/billing');
  assert.equal(crumbs[1]?.href, undefined);
});

test('buildBreadcrumbs routes brand creatives through brands list', () => {
  const brandId = '550e8400-e29b-41d4-a716-446655440002';
  const crumbs = buildBreadcrumbs(`/brand-creatives/${brandId}`, {
    [brandId]: 'Velox Checkout',
  });

  assert.equal(crumbs[0]?.label, 'Brands');
  assert.equal(crumbs[0]?.href, '/brands');
  assert.equal(crumbs[1]?.label, 'Velox Checkout');
  assert.equal(crumbs[1]?.href, undefined);
});

test('buildBreadcrumbs does not link nested report segments without routes', () => {
  const crumbs = buildBreadcrumbs('/reports/ml/score-distribution');

  assert.deepEqual(
    crumbs.map((crumb) => crumb.label),
    ['Reports', 'Ml', 'Score Distribution']
  );
  assert.equal(crumbs[0]?.href, '/reports');
  assert.equal(crumbs[1]?.href, undefined);
  assert.equal(crumbs[2]?.href, undefined);
});

test('buildBreadcrumbs aliases dashboards root to buyer dashboard', () => {
  const crumbs = buildBreadcrumbs('/dashboards/buyer');

  assert.deepEqual(
    crumbs.map((crumb) => ({ label: crumb.label, href: crumb.href })),
    [
      { label: 'Dashboards', href: '/dashboards/buyer' },
      { label: 'Dashboard', href: undefined },
    ]
  );
});

test('buildBreadcrumbs links landers list from editor path', () => {
  const landerId = '550e8400-e29b-41d4-a716-446655440003';
  const crumbs = buildBreadcrumbs(`/landers/${landerId}/editor`, {
    [landerId]: 'Main lander',
  });

  assert.equal(crumbs[0]?.href, '/landers');
  assert.equal(crumbs[1]?.href, undefined);
  assert.equal(crumbs[2]?.href, undefined);
});
