import assert from 'node:assert/strict';
import test from 'node:test';

import { resolveRoutePermission, sessionHasRoutePermission } from '@/lib/route_permissions';

test('resolveRoutePermission maps ops sub-routes to shards:read', () => {
  const rule = resolveRoutePermission('/ops/dlq');
  assert.equal(rule?.permission, 'shards:read');
});

test('resolveRoutePermission maps audit to audit:read', () => {
  const rule = resolveRoutePermission('/audit');
  assert.equal(rule?.permission, 'audit:read');
});

test('resolveRoutePermission maps campaign editor to campaigns read permissions', () => {
  const rule = resolveRoutePermission('/campaigns/550e8400-e29b-41d4-a716-446655440000/edit');
  assert.deepEqual(rule?.permissionAny, ['campaigns:read', 'campaigns:read:masked']);
});

test('sessionHasRoutePermission denies media buyer on ops routes', () => {
  const rule = resolveRoutePermission('/ops');
  assert.equal(sessionHasRoutePermission(['campaigns:read', 'customers:read'], rule), false);
});

test('sessionHasRoutePermission allows operator on ops routes', () => {
  const rule = resolveRoutePermission('/ops');
  assert.equal(sessionHasRoutePermission(['shards:read'], rule), true);
});

test('sessionHasRoutePermission honors wildcard permissions', () => {
  const rule = resolveRoutePermission('/ops');
  assert.equal(sessionHasRoutePermission(['*'], rule), true);
});
