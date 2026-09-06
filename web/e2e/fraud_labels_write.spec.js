import { test, expect } from '@playwright/test';

import {
  ensureCustomerScopeLoaded,
  fetchSessionCustomerId,
  gotoLive,
  integrationFraudLabelReason,
  integrationRunToken,
  isApiGet,
  isApiPost,
  loginAsAdmin,
  mainHeading,
  randomHex32,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('upsert fraud label posts to API and refreshes list', { tag: '@write' }, async ({ page }) => {
  await loginAsAdmin(page);
  const runToken = integrationRunToken();
  const ipHash = randomHex32();
  const reason = integrationFraudLabelReason(runToken);

  await gotoLive(page, '/fraud/labels');
  await expect(mainHeading(page, 'Fraud labels')).toBeVisible();

  const customerRequired = page.getByText('Customer required', { exact: true });
  if (await customerRequired.isVisible({ timeout: 3000 }).catch(() => false)) {
    const loaded = await ensureCustomerScopeLoaded(page, {
      inputSelector: '#labels-customer-id',
      buttonName: 'Load',
      listPathPart: '/api/v1/fraud/labels',
    });
    if (!loaded) {
      test.skip(true, 'integration: no default_customer_id in session');
      return;
    }
  }

  const customerId = await fetchSessionCustomerId(page);
  if (!customerId) {
    test.skip(true, 'integration: no default_customer_id in session');
    return;
  }

  await page.locator('#labels-ip-hash').fill(ipHash);
  await page.locator('#labels-reason').fill(reason);

  const upsertPost = page.waitForResponse(
    (response) =>
      isApiPost('/api/v1/fraud/labels', 200)(response) &&
      response.url().includes(`customer_id=${encodeURIComponent(customerId)}`),
    { timeout: 20_000 }
  );
  const refreshedList = page.waitForResponse(isApiGet('/api/v1/fraud/labels'), { timeout: 20_000 });

  await page.getByRole('button', { name: 'Upsert label' }).click();

  await upsertPost;
  const listResponse = await refreshedList;
  const listBody = await listResponse.json();
  expect(listBody.items).toEqual(
    expect.arrayContaining([expect.objectContaining({ ip_hash: ipHash })])
  );

  await expect(page.getByText('Label saved. List refreshed.', { exact: true })).toBeVisible({
    timeout: 15_000,
  });
  await expect(page.getByRole('cell', { name: ipHash, exact: true })).toBeVisible({
    timeout: 15_000,
  });
});
