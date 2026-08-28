import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import {
  ADMIN_USER,
  mockAuthedSession,
  mockEmptyCampaigns,
  mockPlatformSettings,
  mockReportCatalog,
} from './helpers.js';

test.describe('accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await mockAuthedSession(page, ADMIN_USER);
    await mockEmptyCampaigns(page);
    await mockPlatformSettings(page);
  });

  test('campaigns page has no critical axe violations', async ({ page }) => {
    await page.goto('/campaigns');
    const results = await new AxeBuilder({ page }).analyze();
    expect(results.violations.filter((v) => v.impact === 'critical')).toEqual([]);
  });

  test('billing page has no critical axe violations', async ({ page }) => {
    await page.route('**/api/v1/billing/invoices*', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ items: [], total: 0 }),
      });
    });
    await page.goto('/billing?customer_id=cust-1');
    const results = await new AxeBuilder({ page }).analyze();
    expect(results.violations.filter((v) => v.impact === 'critical')).toEqual([]);
  });

  test('reports hub has no critical axe violations', async ({ page }) => {
    await mockReportCatalog(page, []);
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
});
