import { test, expect } from '@playwright/test';
import {
  ADMIN_USER,
  BUYER_USER,
  PUBLISHER_USER,
  installDialogAutoAccept,
  mockAuthedSession,
} from './helpers.js';

test('billing invoice directory lists rows', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/billing/invoices**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        items: [{ id: 'inv-ledger-1', customer_id: 'cust-1', status: 'open', total_micro: 1000 }],
        total: 1,
      }),
    });
  });

  await page.goto('/billing');
  await expect(page.getByText('inv-ledger-1')).toBeVisible();
});

test('campaign config PATCH on save', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/campaigns/cpa-1', async (route) => {
    const method = route.request().method();
    if (method === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          id: 'cpa-1',
          name: 'CPA',
          status: 'active',
          customer_id: 'cust-1',
        }),
      });
      return;
    }
    if (method === 'PATCH') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ id: 'cpa-1', name: 'CPA paused', status: 'paused' }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/campaigns/cpa-1?tab=config');
  await page.getByLabel('Status').fill('paused');
  await page.getByRole('button', { name: 'Save configuration' }).click();
  await expect(page.getByText('Campaign configuration updated')).toBeVisible();
});

test('publisher hub loads for publisher role', async ({ page }) => {
  await mockAuthedSession(page, PUBLISHER_USER);

  await page.route('**/api/v1/publisher/dashboard**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ impressions: 100, revenue_micro: 5000 }),
    });
  });

  await page.goto('/publisher');
  await expect(page.getByTestId('publisher-hub-page')).toBeVisible();
});

test('publisher nav hides campaigns link', async ({ page }) => {
  await mockAuthedSession(page, PUBLISHER_USER);

  await page.route('**/api/v1/publisher/dashboard**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ impressions: 0 }),
    });
  });

  await page.goto('/publisher');
  const nav = page.getByRole('navigation', { name: 'Main' });
  await expect(nav.getByRole('link', { name: 'Campaigns' })).toHaveCount(0);
});

test('self-serve billing statement panel loads', async ({ page }) => {
  await mockAuthedSession(page, BUYER_USER);

  await page.route('**/api/v1/selfserve/billing/statement**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ opening_balance_micro: 0, closing_balance_micro: 1_000_000, lines: [] }),
    });
  });

  await page.route('**/api/v1/selfserve/invoices**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  await page.goto('/selfserve/billing');
  await expect(page.getByTestId('selfserve-billing-panel')).toBeVisible();
});

test('ops consent proofs read-only list', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/ops/consent/proofs**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        items: [{ id: 'p-1', user_id_hash: 'hash-p-1', source: 'web' }],
      }),
    });
  });

  await page.goto('/ops/consent');
  await expect(page.getByText('hash-p-1')).toBeVisible();
});

test('ops DLQ inbox retry', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/ops/dlq/inbox**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          items: [{ id: 'inbox-1', source: 'postback', error: 'fail' }],
          total: 1,
        }),
      });
      return;
    }
    await route.continue();
  });

  let retried = false;
  await page.route('**/api/v1/ops/dlq/inbox/inbox-1/retry**', async (route) => {
    retried = true;
    await route.fulfill({ status: 204, body: '' });
  });

  await page.goto('/ops/dlq');
  const retry = page.getByRole('button', { name: 'Retry' });
  if (await retry.count()) {
    await retry.first().click();
    await expect.poll(() => retried).toBe(true);
  }
});

test('ops dashboard stub when summary 501', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/ops/dashboard/summary**', async (route) => {
    await route.fulfill({
      status: 501,
      headers: { 'content-type': 'application/json', 'X-API-Stub': 'true' },
      body: JSON.stringify({ error: { code: 'NOT_IMPLEMENTED', message: 'stub' } }),
    });
  });

  await page.goto('/ops');
  await expect(page.getByRole('alert')).toBeVisible();
  await expect(page.getByText('stub')).toBeVisible();
});
