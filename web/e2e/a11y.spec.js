/** harness=mock_api — Playwright route.fulfill; does not prove Go handler or CH/PG. */
import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { mockAuthedSession, ADMIN_USER } from './helpers.js';

const PLATFORM_VIEW = {
  config: {
    tracking_domain: 'track.example',
    default_currency: 'USD',
    timezone: 'UTC',
    ingress_schema: 'ad_event_processor_native',
    telemetry_enabled: true,
    profile: 'single_vps',
    edge_xdp: false,
    network_interface: 'eth0',
    stripe: { enabled: false },
  },
  bootstrap_complete: true,
  restart_required: [],
};

test.describe('accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await mockAuthedSession(page, ADMIN_USER);

    await page.route('**/api/v1/campaigns*', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ items: [], total: 0 }),
      });
    });

    await page.route('**/api/v1/settings/platform', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(PLATFORM_VIEW),
      });
    });

    await page.route('**/api/v1/customers/*/wallet', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          balance_micro: 0,
          allowed_overdraft_micro: 0,
          currency: 'USD',
        }),
      });
    });

    await page.route('**/api/v1/customers/*/balance', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ balance: '0.00', currency: 'USD', ledger: [] }),
      });
    });

    await page.route('**/api/v1/billing/invoices*', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ items: [], total: 0 }),
      });
    });
  });

  test('campaigns page has no critical axe violations', async ({ page }) => {
    await page.goto('/campaigns');
    const results = await new AxeBuilder({ page }).analyze();
    expect(results.violations.filter((v) => v.impact === 'critical')).toEqual([]);
  });

  test('billing page has no critical axe violations', async ({ page }) => {
    await page.goto('/billing?customer_id=cust-1');
    const results = await new AxeBuilder({ page }).analyze();
    expect(results.violations.filter((v) => v.impact === 'critical')).toEqual([]);
  });

  test('reports hub has no critical axe violations', async ({ page }) => {
    await page.route('**/api/v1/reports/**', async (route) => {
      await route.fulfill({
        status: 501,
        headers: { 'content-type': 'application/json', 'X-API-Stub': 'true' },
        body: JSON.stringify({ error: { code: 'NOT_IMPLEMENTED', message: 'stub' } }),
      });
    });
    await page.goto('/reports');
    const results = await new AxeBuilder({ page }).analyze();
    expect(results.violations.filter((v) => v.impact === 'critical')).toEqual([]);
  });

  test('placements report has no critical axe violations', async ({ page }) => {
    await page.route('**/api/v1/reports/placements*', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ items: [], total: 0, freshness: { stale: false } }),
      });
    });
    await page.goto('/reports/placements?customer_id=cust-1');
    const results = await new AxeBuilder({ page }).analyze();
    expect(results.violations.filter((v) => v.impact === 'critical')).toEqual([]);
  });

  test('settings page has no critical axe violations', async ({ page }) => {
    await page.goto('/settings');
    const results = await new AxeBuilder({ page }).analyze();
    expect(results.violations.filter((v) => v.impact === 'critical')).toEqual([]);
  });

  test('confirm dialog has no critical axe violations', async ({ page }) => {
    await page.goto('/settings');
    await page.getByRole('button', { name: 'Save' }).click();
    await expect(page.getByRole('dialog')).toBeVisible();
    const results = await new AxeBuilder({ page })
      .include('[role="dialog"]')
      .analyze();
    expect(results.violations.filter((v) => v.impact === 'critical')).toEqual([]);
  });
});
