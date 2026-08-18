import { test, expect } from '@playwright/test';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const CUSTOMER_ID = '550e8400-e29b-41d4-a716-446655440000';

async function mockOpsOverviewApis(page) {
  await page.route('**/api/v1/ops/doctor', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ overall: 'ok', checks: [] }),
    });
  });

  await page.route('**/api/v1/ops/incidents', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ partial: false, shards: [], outbox: { pending: 0 } }),
    });
  });

  await page.route('**/api/v1/ops/dashboard/summary', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        outbox_pending: 0,
        rps_estimate: 0,
        emergency_breaker: 'closed',
        services: [],
      }),
    });
  });

  await page.route('**/api/v1/ops/rum', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ events: [] }),
    });
  });

  await page.route('**/api/v1/dashboards/operator', async (route) => {
    await route.fulfill({ status: 200, body: JSON.stringify(null) });
  });
}

test('ops billing invariant shows mismatch badge and ledger link', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockOpsOverviewApis(page);

  await page.route('**/api/v1/billing/invariant**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        ok: false,
        customer_id: CUSTOMER_ID,
        balance_micro: 1_000_000,
        ledger_sum_micro: 1_000_002,
        diff_micro: -2,
      }),
    });
  });

  await page.goto('/ops');
  await expect(page.getByTestId('ops-billing-invariant')).toBeVisible();

  await page.getByPlaceholder('Optional customer_id filter').fill(CUSTOMER_ID);
  await page.getByRole('button', { name: 'Check' }).click();

  await expect(page.getByTestId('ops-billing-invariant-mismatch')).toBeVisible();
  await expect(page.locator('.text-danger')).toContainText('-2');
  await expect(page.getByTestId('ops-invariant-billing-link')).toBeVisible();
  await expect(page.getByTestId('ops-invariant-ledger-link')).toHaveAttribute(
    'href',
    `/billing?customer_id=${CUSTOMER_ID}&tab=ledger`
  );
});

test('ops billing invariant shows OK badge when balanced', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockOpsOverviewApis(page);

  await page.route('**/api/v1/billing/invariant**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        ok: true,
        customer_id: CUSTOMER_ID,
        balance_micro: 5_000_000,
        ledger_sum_micro: 5_000_000,
        diff_micro: 0,
      }),
    });
  });

  await page.goto('/ops');
  await expect(page.getByTestId('ops-billing-invariant')).toBeVisible();
  await page.getByRole('button', { name: 'Check' }).click();

  await expect(page.getByTestId('ops-billing-invariant-ok')).toBeVisible();
  await expect(page.getByTestId('ops-billing-invariant-mismatch')).toHaveCount(0);
});

test('ops billing invariant shows fleet scan cap hint', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);
  await mockOpsOverviewApis(page);

  await page.route('**/api/v1/billing/invariant**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        ok: true,
        fleet_scan_limit: 500,
      }),
    });
  });

  await page.goto('/ops');
  await page.getByRole('button', { name: 'Check' }).click();

  await expect(page.getByTestId('ops-billing-invariant-fleet-cap')).toContainText('500');
});
