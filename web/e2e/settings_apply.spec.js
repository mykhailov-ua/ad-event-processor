import { test, expect } from '@playwright/test';

import {
  gotoLive,
  isApiGet,
  isApiPost,
  loginAsAdmin,
  mainHeading,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('settings POST apply returns written_path', { tag: '@write' }, async ({ page }) => {
  await loginAsAdmin(page);
  const initialGet = page.waitForResponse(isApiGet('/api/v1/settings/platform'), {
    timeout: 20_000,
  });
  await gotoLive(page, '/settings');
  await expect(mainHeading(page, 'Settings')).toBeVisible();

  const initialResponse = await initialGet;
  const initialBody = await initialResponse.json();
  if (!initialBody.bootstrap_complete) {
    test.skip(true, 'integration: platform bootstrap incomplete');
    return;
  }

  page.once('dialog', (dialog) => {
    expect(dialog.message()).toContain('install.compose.env');
    void dialog.accept();
  });

  const applyResponse = page.waitForResponse(isApiPost('/api/v1/settings/platform/apply'), {
    timeout: 20_000,
  });
  await page.getByRole('button', { name: 'Write install.compose.env' }).click();

  const applied = await applyResponse;
  const applyBody = await applied.json();
  expect(applyBody.written_path).toBeTruthy();
  await expect(page.getByText(applyBody.written_path, { exact: false })).toBeVisible({
    timeout: 15_000,
  });
});
