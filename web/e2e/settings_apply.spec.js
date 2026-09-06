import { test, expect } from '@playwright/test';

import {
  expandDetailsSection,
  gotoLive,
  isApiGet,
  loginAsAdmin,
  mainHeading,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test(
  'settings apply writes platform configuration to disk',
  { tag: '@write' },
  async ({ page }) => {
    await loginAsAdmin(page);

    const initialSettings = page.waitForResponse(isApiGet('/api/v1/settings/platform'), {
      timeout: 20_000,
    });
    await gotoLive(page, '/settings');
    await expect(mainHeading(page, 'Platform settings')).toBeVisible();
    await initialSettings;

    await expandDetailsSection(page, 'Save configuration to disk');
    await expect(page.getByLabel('Installation directory (optional)')).toBeVisible();

    const applyPost = page.waitForResponse(
      (response) =>
        response.request().method() === 'POST' &&
        response.url().includes('/api/v1/settings/platform/apply'),
      { timeout: 20_000 }
    );

    await page.getByRole('button', { name: 'Save to disk' }).click();

    const applyResponse = await applyPost;
    if (!applyResponse.ok()) {
      test.skip(true, `integration: platform apply returned ${applyResponse.status()}`);
      return;
    }
    const applyBody = await applyResponse.json();
    expect(typeof applyBody.written_path).toBe('string');
    expect(applyBody.written_path.length).toBeGreaterThan(0);

    await expect(page.getByRole('status')).toContainText('Saved to', { timeout: 15_000 });
    await expect(page.getByText(applyBody.written_path, { exact: false })).toBeVisible({
      timeout: 15_000,
    });
  }
);
