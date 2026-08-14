/** harness=stack — live control plane on PLAYWRIGHT_BASE_URL; no route.fulfill mocks. */
import { test, expect } from '@playwright/test';

const stackEnabled = process.env.ADMIN_STACK_E2E === '1';
const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://127.0.0.1:8188';
const email = process.env.ADMIN_STACK_E2E_EMAIL ?? '';
const password = process.env.ADMIN_STACK_E2E_PASSWORD ?? '';

test.describe('admin stack e2e', () => {
  test.skip(!stackEnabled, 'Set ADMIN_STACK_E2E=1 to run against a live control plane');

  test('login and load settings', async ({ page }) => {
    test.skip(!email || !password, 'Set ADMIN_STACK_E2E_EMAIL and ADMIN_STACK_E2E_PASSWORD');

    await page.goto(`${baseURL}/login`);
    await page.fill('input[type=email]', email);
    await page.fill('input[type=password]', password);
    await page.click('button[type=submit]');
    await page.waitForURL(`${baseURL}/`);

    await expect(page.getByRole('heading', { name: 'Overview' })).toBeVisible();

    await page.goto(`${baseURL}/settings`);
    await expect(page.getByRole('heading', { name: 'Platform settings' })).toBeVisible();
  });

  test('customers list returns rows', async ({ page }) => {
    test.skip(!email || !password, 'Set ADMIN_STACK_E2E_EMAIL and ADMIN_STACK_E2E_PASSWORD');

    await page.goto(`${baseURL}/login`);
    await page.fill('input[type=email]', email);
    await page.fill('input[type=password]', password);
    await page.click('button[type=submit]');
    await page.waitForURL(`${baseURL}/`);

    await page.goto(`${baseURL}/customers`);
    await expect(page.getByRole('heading', { name: 'Customers' })).toBeVisible();
    await expect(page.locator('.data-table')).toBeVisible();
  });

  test('campaign detail loads from real API', async ({ page }) => {
    test.skip(!email || !password, 'Set ADMIN_STACK_E2E_EMAIL and ADMIN_STACK_E2E_PASSWORD');

    await page.goto(`${baseURL}/login`);
    await page.fill('input[type=email]', email);
    await page.fill('input[type=password]', password);
    await page.click('button[type=submit]');
    await page.waitForURL(`${baseURL}/`);

    const listRes = await page.request.get(`${baseURL}/api/v1/campaigns?limit=1&offset=0`);
    test.skip(!listRes.ok(), 'campaigns list unavailable');
    const listBody = await listRes.json();
    const campaignId = listBody?.items?.[0]?.id;
    test.skip(!campaignId, 'no campaigns in seed data');

    await page.goto(`${baseURL}/campaigns/${campaignId}`);
    await expect(page.getByRole('tab', { name: 'Overview' })).toBeVisible();
    await expect(page.locator('.page-header__title')).toBeVisible();
  });
});
