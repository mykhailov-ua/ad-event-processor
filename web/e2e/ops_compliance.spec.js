import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

test('consent proof browser lists read-only proofs', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/ops/consent/proofs**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        items: [
          {
            id: 42,
            user_id_hash: 'abc123',
            purposes: 1,
            source: 'cmp',
            recorded_at: '2026-08-12T10:00:00Z',
            ad_storage: true,
            analytics_storage: false,
          },
        ],
      }),
    });
  });

  await page.goto('/ops/consent');
  await expect(page.getByTestId('consent-browser')).toBeVisible();
  await expect(page.getByTestId('consent-proof-row-42')).toContainText('cmp');
  await expect(page.getByRole('button', { name: 'Delete' })).toHaveCount(0);
});

test('support feedback submits POST /support/feedback', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  let feedbackPosted = false;

  await page.route('**/api/v1/support/feedback/meta', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ deployment_id: 'dep-e2e', binary_version: 'e2e' }),
    });
  });

  await page.route('**/api/v1/support/feedback', async (route) => {
    if (route.request().method() === 'POST') {
      feedbackPosted = true;
      const body = route.request().postDataJSON();
      expect(body.contact_email).toBe('ops@example.com');
      expect(body.message).toBe('dashboard slow');
      await route.fulfill({
        status: 201,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ id: 'fb-e2e-1' }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/support/feedback');
  await expect(page.getByTestId('feedback-meta')).toContainText('dep-e2e');
  await page.getByTestId('feedback-email').fill('ops@example.com');
  await page.getByTestId('feedback-message').fill('dashboard slow');
  await page.getByTestId('feedback-submit').click();
  await expect.poll(() => feedbackPosted).toBe(true);
  await expect(page.getByTestId('feedback-last-id')).toContainText('fb-e2e-1');
});

test('domains TLS check calls tls-allowed endpoint', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  let tlsChecked = false;

  await page.route('**/api/v1/domains', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify([]),
    });
  });

  await page.route('**/api/v1/ops/domains/**/tls-allowed', async (route) => {
    tlsChecked = true;
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ allowed: true }),
    });
  });

  await page.goto('/settings/domains');
  await page.getByTestId('tls-check-host').fill('buyer.example.com');
  await page.getByTestId('tls-check-submit').click();
  await expect.poll(() => tlsChecked).toBe(true);
  await expect(page.getByTestId('tls-check-result')).toContainText('allowed');
});

test('ops danger zone reload RBAC via POST /ops/roles/reload', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  let rolesReloaded = false;

  await page.route('**/api/v1/ops/incidents', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ partial: false, shards: [], outbox: { pending: 0 } }),
    });
  });

  await page.route('**/api/v1/ops/doctor', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ overall: 'ok', checks: [] }),
    });
  });

  await page.route('**/api/v1/ops/dashboard/summary', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ outbox_pending: 0, emergency_breaker: 'closed', services: [] }),
    });
  });

  await page.route('**/api/v1/ops/roles/reload', async (route) => {
    rolesReloaded = true;
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ status: 'reloaded', path: '/etc/operator/roles.yaml' }),
    });
  });

  await page.goto('/ops');
  await page.getByTestId('roles-reload').click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByLabel('Type DELETE to confirm').fill('DELETE');
  await page.getByRole('checkbox', { name: 'I understand the consequences' }).check();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();
  await expect.poll(() => rolesReloaded).toBe(true);
});

test('unified DLQ inbox retries postback source', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  let retryCalled = false;

  await page.route('**/api/v1/ops/dlq/inbox**', async (route) => {
    if (route.request().method() !== 'GET') {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        items: [
          {
            id: '7',
            source: 'postback',
            campaign_id: 'camp-1',
            event_type: 'conversion',
            error: 'timeout',
            failed_at: '2026-08-12T10:00:00Z',
            status: 'FAILED',
          },
        ],
      }),
    });
  });

  await page.route('**/api/v1/ops/dlq/inbox/7/retry', async (route) => {
    retryCalled = true;
    const body = route.request().postDataJSON();
    expect(body.source).toBe('postback');
    await route.fulfill({ status: 202, body: '' });
  });

  await page.goto('/ops/dlq');
  await expect(page.getByTestId('ops-dlq-inbox')).toBeVisible();
  await page.getByTestId('dlq-inbox-retry-postback-7').click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();
  await expect.poll(() => retryCalled).toBe(true);
});
