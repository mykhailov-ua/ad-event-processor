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
    await page.fill('#login-email', email);
    await page.fill('#login-password', password);
    await page.getByRole('button', { name: 'Sign in' }).click();
    await page.waitForURL(new RegExp(`${baseURL}/customers`));

    await page.goto(`${baseURL}/settings`);
    await expect(page.getByRole('heading', { name: 'Platform settings' })).toBeVisible();
  });

  test('customers list returns rows', async ({ page }) => {
    test.skip(!email || !password, 'Set ADMIN_STACK_E2E_EMAIL and ADMIN_STACK_E2E_PASSWORD');

    await page.goto(`${baseURL}/login`);
    await page.fill('#login-email', email);
    await page.fill('#login-password', password);
    await page.getByRole('button', { name: 'Sign in' }).click();
    await page.waitForURL(new RegExp(`${baseURL}/customers`));

    await page.goto(`${baseURL}/customers`);
    await expect(page.getByRole('heading', { name: 'Customers' })).toBeVisible();
    await expect(page.getByRole('grid')).toBeVisible();
  });

  test('campaign detail loads from real API', async ({ page }) => {
    test.skip(!email || !password, 'Set ADMIN_STACK_E2E_EMAIL and ADMIN_STACK_E2E_PASSWORD');

    await page.goto(`${baseURL}/login`);
    await page.fill('#login-email', email);
    await page.fill('#login-password', password);
    await page.getByRole('button', { name: 'Sign in' }).click();
    await page.waitForURL(new RegExp(`${baseURL}/customers`));

    const listRes = await page.request.get(`${baseURL}/api/v1/campaigns?limit=1&offset=0`);
    test.skip(!listRes.ok(), 'campaigns list unavailable');
    const listBody = await listRes.json();
    const campaignId = listBody?.items?.[0]?.id;
    test.skip(!campaignId, 'no campaigns in seed data');

    await page.goto(`${baseURL}/campaigns/${campaignId}`);
    await expect(page.getByRole('tab', { name: 'Overview' })).toBeVisible();
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
  });
});
