import { test, expect } from '@playwright/test';
import { mockAuthedSession } from './helpers.js';

const RTB_USER = {
  id: 'rtb-floors-1',
  email: 'rtb-floors@test.local',
  role: 'A',
  customer_id: '',
  permissions: ['rtb:read', 'settings:write', 'campaigns:read'],
};

test('RTB floors preview and apply use confirm on apply', async ({ page }) => {
  await mockAuthedSession(page, RTB_USER);

  await page.route('**/api/v1/rtb/integration-profile', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ openrtb_version: '2.6', supported: [], required: [] }),
    });
  });

  await page.route('**/api/v1/rtb/shadow-diff**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ source: 'unavailable' }),
    });
  });

  let applyDryRun = null;
  await page.route('**/api/v1/rtb/floors/apply**', async (route) => {
    const url = route.request().url();
    applyDryRun = url.includes('dry_run=true');
    const dryRun = applyDryRun;
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        dry_run: dryRun,
        applied: dryRun ? 0 : 1,
        outbox_rows: dryRun ? 0 : 2,
        suggestions: dryRun
          ? [
              {
                placement_id: 'plc-1',
                deal_id: 'deal-1',
                current_floor_micro: 1000,
                suggested_floor_micro: 1500,
                win_rate: 0.42,
                sample_n: 100,
                floor_bucket_micro: 1000,
                computed_at: '2026-08-12T10:00:00Z',
              },
            ]
          : [],
      }),
    });
  });

  await page.goto('/rtb/integration');
  await expect(page.getByTestId('rtb-floors-preview')).toBeVisible();
  await page.getByTestId('rtb-floors-preview').click();
  await expect.poll(() => applyDryRun).toBe(true);
  await expect(page.getByText('plc-1')).toBeVisible();

  applyDryRun = null;
  await page.getByTestId('rtb-floors-apply').click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.getByRole('dialog').getByRole('button', { name: 'Confirm' }).click();
  await expect.poll(() => applyDryRun).toBe(false);
});

test('RTB reconcile export downloads JSON summary', async ({ page }) => {
  await mockAuthedSession(page, RTB_USER);

  await page.route('**/api/v1/rtb/integration-profile', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ openrtb_version: '2.6', supported: [], required: [] }),
    });
  });

  await page.route('**/api/v1/rtb/shadow-diff**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ source: 'unavailable' }),
    });
  });

  await page.route('**/api/v1/rtb/reconcile/export**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        window: '24h0m0s',
        bids: 1200,
        wins: 340,
        spend_micro: 5_000_000,
        live_gate_ready: true,
        shadow: { source: 'unavailable' },
      }),
    });
  });

  await page.goto('/rtb/integration');
  await page.getByTestId('rtb-reconcile-export').click();
  await expect(page.getByTestId('rtb-reconcile-summary')).toBeVisible();
  await expect(
    page.getByTestId('rtb-reconcile-summary').getByText('1200', { exact: true })
  ).toBeVisible();
});
