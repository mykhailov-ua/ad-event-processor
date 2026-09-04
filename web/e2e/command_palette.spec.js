import { test, expect } from '@playwright/test';

import {
  gotoLive,
  isApiGet,
  loginAsAdmin,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('command palette lists routes from GET /api/v1/command-palette/routes', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoLive(page, '/customers');

  const routesResponse = page.waitForResponse(isApiGet('/api/v1/command-palette/routes'), {
    timeout: 20_000,
  });
  await page.keyboard.press('Control+k');
  await expect(page.getByRole('dialog')).toBeVisible();
  await expect(page.getByLabel('Command palette search')).toBeVisible();

  const forbidden = page.getByText('forbidden', { exact: false });
  if (await forbidden.isVisible({ timeout: 3000 }).catch(() => false)) {
    test.skip(true, 'command palette forbidden for session permissions');
  }

  const response = await routesResponse;
  const body = await response.json();
  expect(Array.isArray(body.items)).toBe(true);
  expect(body.items.length).toBeGreaterThan(0);

  const first = body.items[0];
  expect(typeof first.label).toBe('string');
  await expect(page.getByRole('option', { name: first.label })).toBeVisible({ timeout: 15_000 });
});
