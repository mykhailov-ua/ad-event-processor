import { test, expect } from '@playwright/test';

import {
  expectErrorBlockVisible,
  gotoLive,
  gotoLiveAwaitGet,
  isApiGet,
  loginAsAdmin,
  mainHeading,
  OPS_SECTION_READS,
  skipUnlessIntegrationReady,
  stubApiRoute,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('ops home shows section navigation and loads GET /api/v1/ops/home', async ({ page }) => {
  await loginAsAdmin(page);
  const homeGet = await gotoLiveAwaitGet(page, '/ops', '/api/v1/ops/home');
  await expect(mainHeading(page, 'Ops')).toBeVisible();
  await expect(page.getByRole('navigation', { name: 'Ops sections' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Incidents' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Metrics' })).toBeVisible();
  expect(homeGet.ok()).toBe(true);
});

test('ops metrics page shows live toggle', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoLiveAwaitGet(page, '/ops/metrics', '/api/v1/ops/dashboard/metrics');
  await expect(page.getByRole('heading', { name: 'Dashboard metrics' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Live', exact: true })).toBeVisible();
});

test(
  'ops home shows ErrorBlock when GET /api/v1/ops/home returns 500',
  { tag: '@L3' },
  async ({ page }) => {
    await loginAsAdmin(page);
    await stubApiRoute(page, '/api/v1/ops/home', 500, {
      error: { code: 'INTERNAL_ERROR', message: 'ops home unavailable' },
    });
    const homeResponse = page.waitForResponse(isApiGet('/api/v1/ops/home', 500), {
      timeout: 20_000,
    });
    await gotoLive(page, '/ops');
    await homeResponse;
    await expect(mainHeading(page, 'Ops')).toBeVisible();
    await expectErrorBlockVisible(page, 'Could not load ops snapshot');
  }
);

test('ops home shows reload roles control', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoLiveAwaitGet(page, '/ops', '/api/v1/ops/home');
  await expect(page.getByRole('button', { name: 'Reload roles' })).toBeVisible();
});

test('ops section links resolve without 404 and load read endpoints', async ({ page }) => {
  await loginAsAdmin(page);

  for (const { path, api } of OPS_SECTION_READS) {
    const sectionGet = page.waitForResponse(isApiGet(api), { timeout: 20_000 });
    await page.goto(`${path}`);
    await expect(page.getByRole('navigation', { name: 'Ops sections' })).toBeVisible();
    await expect(page.getByText('404')).toHaveCount(0);
    await expect(page.getByText('Page not found')).toHaveCount(0);
    const response = await sectionGet;
    expect(response.ok()).toBe(true);
  }
});
