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

test('moderator corpus page lists tuples and upserts JA3', { tag: '@write' }, async ({ page }) => {
  await loginAsAdmin(page);
  await gotoLive(page, '/fraud/moderator-corpus');
  await expect(mainHeading(page, 'Moderator corpus')).toBeVisible();

  const ja3 = `771,4865-${Date.now()}`;

  const listGet = page.waitForResponse(isApiGet('/api/v1/fraud/moderator-corpus'), {
    timeout: 20_000,
  });
  await listGet;

  const upsertPost = page.waitForResponse(isApiPost('/api/v1/fraud/moderator-corpus', 200), {
    timeout: 20_000,
  });
  const refreshedList = page.waitForResponse(isApiGet('/api/v1/fraud/moderator-corpus'), {
    timeout: 20_000,
  });

  await page.getByPlaceholder('JA3 (required)').fill(ja3);
  await page.getByRole('button', { name: 'Save tuple' }).click();

  await upsertPost;
  await refreshedList;

  await expect(page.getByText('Corpus tuple saved', { exact: true })).toBeVisible({
    timeout: 15_000,
  });
  await expect(page.getByRole('cell', { name: ja3, exact: true })).toBeVisible({
    timeout: 15_000,
  });
});
