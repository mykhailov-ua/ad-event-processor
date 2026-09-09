import test from 'node:test';
import assert from 'node:assert/strict';

import {
  CONTROL_PLANE_CORE_NAV_PATHS,
  CONTROL_PLANE_OPERATIONS_NAV_PATHS,
  controlPlaneCampaignReportPath,
  filterControlPlaneNavGroups,
} from './control_plane_scope.ts';

test('control plane nav paths cover CP surfaces', () => {
  assert.deepEqual(CONTROL_PLANE_CORE_NAV_PATHS, [
    '/customers',
    '/campaigns',
    '/team',
    '/settings',
  ]);
  assert.deepEqual(CONTROL_PLANE_OPERATIONS_NAV_PATHS, [
    '/ops',
    '/audit',
    '/exports',
    '/integrations',
  ]);
});

test('controlPlaneCampaignReportPath is hidden in CP mode', () => {
  assert.equal(controlPlaneCampaignReportPath('camp-1'), null);
});

test('filterControlPlaneNavGroups respects permissions', () => {
  const groups = filterControlPlaneNavGroups(['customers:read', 'campaigns:read']);
  const paths = groups.flatMap((group) => group.items.map((item) => item.path));
  assert.ok(paths.includes('/customers'));
  assert.ok(paths.includes('/campaigns'));
  assert.ok(!paths.includes('/ops'));
});
