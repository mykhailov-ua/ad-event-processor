import assert from 'node:assert/strict';
import test from 'node:test';

import { resolveDevMockRequest } from './handler.ts';
import { DEV_MOCK_CUSTOMERS } from './fixtures.ts';
import { resetDevMockStore } from './store.ts';

test('dev mock lists campaign list facets for customer scope', () => {
  resetDevMockStore();
  const customerId = DEV_MOCK_CUSTOMERS[0].id;
  const facets = resolveDevMockRequest(
    `/api/v1/campaigns/list-facets?customer_id=${encodeURIComponent(customerId)}`,
  );
  assert.equal(facets?.status, 200);
  const body = facets?.body as {
    countries: string[];
    owners: Array<{ user_id: string; email?: string }>;
  };
  assert.ok(body.countries.length > 0);
  assert.ok(body.owners.length > 0);
  assert.ok(body.owners.every((owner) => owner.user_id));
});

test('dev mock lists campaigns with filters', () => {
  resetDevMockStore();
  const all = resolveDevMockRequest('/api/v1/campaigns?limit=50&offset=0');
  assert.equal(all?.status, 200);
  const body = all?.body as {
    items: unknown[];
    total: number;
    filters_applied?: Record<string, string>;
    status_totals?: { active: number; paused: number; archived: number; total: number };
  };
  assert.ok(body.total >= 20);
  assert.equal(body.items.length, body.total);
  assert.equal(body.filters_applied, undefined);
  assert.ok(body.status_totals);
  assert.equal(body.status_totals?.total, body.total);

  const active = resolveDevMockRequest(
    `/api/v1/campaigns?customer_id=${encodeURIComponent(DEV_MOCK_CUSTOMERS[0].id)}&status=ACTIVE&limit=50`,
  );
  const activeBody = active?.body as {
    items: { status: string }[];
    filters_applied?: Record<string, string>;
    status_totals?: { active: number; total: number };
  };
  assert.ok(activeBody.items.every((row) => row.status === 'ACTIVE'));
  assert.equal(activeBody.filters_applied?.status, 'ACTIVE');
  assert.equal(activeBody.filters_applied?.customer_id, DEV_MOCK_CUSTOMERS[0].id);
  assert.ok(activeBody.status_totals);
  assert.equal(activeBody.status_totals?.active, activeBody.items.length);
});

test('dev mock ops home returns composite snapshot', () => {
  const home = resolveDevMockRequest('/api/v1/ops/home');
  assert.equal(home?.status, 200);
  const body = home?.body as {
    doctor: { checks: unknown[] };
    stackHealth: { status: string };
    dashboardSummary: { services: unknown[] };
  };
  assert.ok(Array.isArray(body.doctor.checks));
  assert.equal(body.stackHealth.status, 'ok');
  assert.ok(body.dashboardSummary.services.length > 0);
});

test('dev mock session bootstrap is authenticated', () => {
  const boot = resolveDevMockRequest('/api/v1/session/bootstrap');
  assert.equal(boot?.status, 200);
  const body = boot?.body as { user: { email?: string }; session: { role?: string } };
  assert.equal(body.user.email, 'operator@dev.local');
  assert.equal(body.session.role, 'admin');
});

test('dev mock team members use user_id wire field', () => {
  const members = resolveDevMockRequest('/api/v1/team/members?customer_id=test');
  assert.equal(members?.status, 200);
  const body = members?.body as { items: { user_id?: string; id?: string; email?: string }[] };
  assert.ok(body.items.length > 0);
  for (const row of body.items) {
    assert.ok(row.user_id);
    assert.equal(row.id, undefined);
  }
});

test('dev mock wizard session create and commit', () => {
  const templates = resolveDevMockRequest('/api/v1/campaigns/onboarding-templates');
  assert.equal(templates?.status, 200);
  const templateRows = templates?.body as { key: string }[];
  assert.ok(templateRows.length >= 1);

  const created = resolveDevMockRequest('/api/v1/campaigns/wizard/session', {
    method: 'POST',
    body: JSON.stringify({
      action: 'create',
      customer_id: DEV_MOCK_CUSTOMERS[0].id,
      template_key: templateRows[0].key,
    }),
  });
  assert.equal(created?.status, 201);
  const session = created?.body as { session_id: string; current_step: string };
  assert.ok(session.session_id);
  assert.equal(session.current_step, 'traffic_source');
});

test('dev mock campaign metrics echoes range and scales clicks by window', () => {
  resetDevMockStore();
  const listed = resolveDevMockRequest('/api/v1/campaigns?limit=1&offset=0');
  const campaignId = (listed?.body as { items: { id: string }[] }).items[0]?.id;
  assert.ok(campaignId);

  const week = resolveDevMockRequest(
    `/api/v1/campaigns/metrics?ids=${encodeURIComponent(campaignId)}&from=2026-02-01T00:00:00.000Z&to=2026-02-08T00:00:00.000Z`,
  );
  const month = resolveDevMockRequest(
    `/api/v1/campaigns/metrics?ids=${encodeURIComponent(campaignId)}&from=2026-02-01T00:00:00.000Z&to=2026-03-01T00:00:00.000Z`,
  );
  assert.equal(week?.status, 200);
  assert.equal(month?.status, 200);

  const weekBody = week?.body as {
    from: string;
    to: string;
    items: Record<string, { clicks?: number }>;
  };
  const monthBody = month?.body as {
    items: Record<string, { clicks?: number }>;
  };
  assert.equal(weekBody.from, '2026-02-01T00:00:00.000Z');
  assert.equal(weekBody.to, '2026-02-08T00:00:00.000Z');
  const weekClicks = weekBody.items[campaignId]?.clicks ?? 0;
  const monthClicks = monthBody.items[campaignId]?.clicks ?? 0;
  assert.ok(weekClicks > 0);
  assert.ok(monthClicks > weekClicks);
});

test('dev mock campaign editor path', () => {
  resetDevMockStore();
  const listed = resolveDevMockRequest('/api/v1/campaigns?limit=1&offset=0');
  const campaignId = (listed?.body as { items: { id: string }[] }).items[0]?.id;
  assert.ok(campaignId);

  const editor = resolveDevMockRequest(`/api/v1/campaigns/${encodeURIComponent(campaignId)}/editor`);
  assert.equal(editor?.status, 200);

  const body = editor?.body as {
    campaign_id: string;
    sections: unknown[];
    completion_pct: number;
  };
  assert.equal(body.campaign_id, campaignId);
  assert.ok(body.sections.length > 0);
  assert.ok(body.completion_pct >= 0);
});

test('dev mock clone-preview and owner assignment', () => {
  resetDevMockStore();
  const listed = resolveDevMockRequest('/api/v1/campaigns?limit=1&offset=0');
  const campaignId = (listed?.body as { items: { id: string }[] }).items[0]?.id;
  assert.ok(campaignId);

  const preview = resolveDevMockRequest(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/clone-preview`,
    {
      method: 'POST',
      body: JSON.stringify({ name_suffix: ' copy' }),
    },
  );
  assert.equal(preview?.status, 200);
  const previewBody = preview?.body as { source_id: string; name: string; would_create: object };
  assert.equal(previewBody.source_id, campaignId);
  assert.match(previewBody.name, / copy$/);
  assert.ok(previewBody.would_create);

  const owner = resolveDevMockRequest(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/owner`,
    {
      method: 'PUT',
      body: JSON.stringify({ user_id: 'buyer-1' }),
    },
  );
  assert.equal(owner?.status, 200);

  const campaign = resolveDevMockRequest(`/api/v1/campaigns/${encodeURIComponent(campaignId)}`);
  const campaignBody = campaign?.body as { owner_user_id?: string };
  assert.equal(campaignBody.owner_user_id, 'buyer-1');
});

test('dev mock legacy ops dlq list and retry', () => {
  const list = resolveDevMockRequest('/api/v1/ops/dlq?limit=25');
  assert.equal(list?.status, 200);
  const listBody = list?.body as { items: unknown[]; partial?: boolean };
  assert.ok(Array.isArray(listBody.items));
  assert.equal(listBody.partial, false);

  const retry = resolveDevMockRequest('/api/v1/ops/dlq/entry-1/retry', { method: 'POST' });
  assert.equal(retry?.status, 202);
});

test('dev mock consent record returns 204', () => {
  const response = resolveDevMockRequest('/api/v1/consent', {
    method: 'POST',
    body: JSON.stringify({
      user_id: 'user-1',
      purposes: 1,
      source: 'admin',
    }),
  });
  assert.equal(response?.status, 204);
  assert.equal(response?.body, undefined);
});
