import assert from 'node:assert/strict';
import test from 'node:test';

import {
  formatRoutePermissionRequirement,
  resolveRoutePermission,
  sessionHasRoutePermission,
} from '@/lib/route_permissions';

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

test('sessionHasRoutePermission denies guarded routes while permissions are undefined', () => {
  const rule = resolveRoutePermission('/ops');
  assert.equal(sessionHasRoutePermission(undefined, rule), false);
});

test('sessionHasRoutePermission allows operator on ops routes', () => {
  const rule = resolveRoutePermission('/ops');
  assert.equal(sessionHasRoutePermission(['shards:read'], rule), true);
});

test('sessionHasRoutePermission honors wildcard permissions', () => {
  const rule = resolveRoutePermission('/ops');
  assert.equal(sessionHasRoutePermission(['*'], rule), true);
});

test('resolveRoutePermission maps dashboards sub-routes to campaigns read permissions', () => {
  const rule = resolveRoutePermission('/dashboards/adops');
  assert.deepEqual(rule?.permissionAny, ['campaigns:read', 'campaigns:read:masked']);
});

test('resolveRoutePermission maps alerts to campaigns:read', () => {
  const rule = resolveRoutePermission('/alerts');
  assert.equal(rule?.permission, 'campaigns:read');
});

test('resolveRoutePermission maps export schedules to campaigns:read', () => {
  const rule = resolveRoutePermission('/exports/schedules');
  assert.equal(rule?.permission, 'campaigns:read');
});

test('resolveRoutePermission maps team to team:read or campaigns:read from nav', () => {
  const rule = resolveRoutePermission('/team');
  assert.deepEqual(rule?.permissionAny, ['team:read', 'campaigns:read']);
});

test('resolveRoutePermission maps disputes to customers:read', () => {
  const rule = resolveRoutePermission('/disputes');
  assert.equal(rule?.permission, 'customers:read');
});

test('formatRoutePermissionRequirement formats single and any permissions', () => {
  assert.equal(formatRoutePermissionRequirement(resolveRoutePermission('/ops')), 'shards:read');
  assert.equal(
    formatRoutePermissionRequirement(resolveRoutePermission('/team')),
    'team:read or campaigns:read'
  );
  assert.equal(formatRoutePermissionRequirement(null), undefined);
});
