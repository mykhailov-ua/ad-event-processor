import { test, expect } from '@playwright/test';

import {
  gotoLive,
  isApiGet,
  isApiPatch,
  loginAsAdmin,
  mainHeading,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('settings PATCH tracking_domain updates platform view', { tag: '@write' }, async ({ page }) => {
  await loginAsAdmin(page);
  const initialGet = page.waitForResponse(isApiGet('/api/v1/settings/platform'), {
    timeout: 20_000,
  });
  await gotoLive(page, '/settings');
  await expect(mainHeading(page, 'Settings')).toBeVisible();

  const initialResponse = await initialGet;
  const initialBody = await initialResponse.json();
  const originalDomain = initialBody.config?.tracking_domain ?? '';
  const nextDomain = originalDomain === 'e2e-settings.example.com'
    ? 'e2e-settings-alt.example.com'
    : 'e2e-settings.example.com';

  await page.locator('#settings-tracking-domain').fill(nextDomain);

  const patchResponse = page.waitForResponse(isApiPatch('/api/v1/settings/platform'), {
    timeout: 20_000,
  });
  await page.getByRole('button', { name: 'Save changes' }).click();

  const patched = await patchResponse;
  const patchRequest = patched.request().postDataJSON();
  expect(patchRequest).toMatchObject({ tracking_domain: nextDomain });

  const patchedBody = await patched.json();
  expect(patchedBody.config?.tracking_domain).toBe(nextDomain);

  if (originalDomain !== nextDomain) {
    await page.locator('#settings-tracking-domain').fill(originalDomain);
    const restorePatch = page.waitForResponse(isApiPatch('/api/v1/settings/platform'), {
      timeout: 20_000,
    });
    await page.getByRole('button', { name: 'Save changes' }).click();
    const restored = await restorePatch;
    const restoredBody = await restored.json();
    expect(restoredBody.config?.tracking_domain).toBe(originalDomain);
  }
});
