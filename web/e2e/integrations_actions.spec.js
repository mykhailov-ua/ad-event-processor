import { test, expect } from '@playwright/test';

import {
  baseURL,
  ensureIntegrationCustomerScope,
  fetchFirstCampaignId,
  fetchSessionCustomerId,
  gotoLive,
  isApiGet,
  isApiPost,
  loginAsAdmin,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('cost sync run posts to API and refreshes snapshot', async ({ page }) => {
  await loginAsAdmin(page);

  const snapshotGet = page.waitForResponse(isApiGet('/api/v1/cost-sync/snapshot'), {
    timeout: 20_000,
  });
  await gotoLive(page, '/integrations/cost-sync');
  await snapshotGet;

  const customerId = await fetchSessionCustomerId(page);
  if (!customerId) {
    test.skip(true, 'integration: no default_customer_id in session');
    return;
  }

  const scoped = await ensureIntegrationCustomerScope(page, '/api/v1/cost-sync/snapshot');
  if (!scoped) {
    test.skip(true, 'integration: could not apply customer scope');
    return;
  }

  const runPost = page.waitForResponse(isApiPost('/api/v1/cost-sync/run', 202), {
    timeout: 20_000,
  });
  const refreshedSnapshot = page.waitForResponse(isApiGet('/api/v1/cost-sync/snapshot'), {
    timeout: 20_000,
  });

  await page.getByRole('button', { name: 'Run sync', exact: true }).click();

  const runResponse = await runPost;
  const runBody = await runResponse.json();
  expect(runBody).toBeTruthy();

  await refreshedSnapshot;

  await expect(
    page.getByText('Sync accepted. Refresh history for results.', { exact: true }),
  ).toBeVisible({ timeout: 15_000 });
});

test('platform campaign sync posts to API', async ({ page }) => {
  await loginAsAdmin(page);

  const linksGet = page.waitForResponse(isApiGet('/api/v1/platform-campaigns/links'), {
    timeout: 20_000,
  });
  await gotoLive(page, '/integrations/platform-campaigns');
  await linksGet;

  const customerId = await fetchSessionCustomerId(page);
  if (!customerId) {
    test.skip(true, 'integration: no default_customer_id in session');
    return;
  }

  const scoped = await ensureIntegrationCustomerScope(page, '/api/v1/platform-campaigns/links');
  if (!scoped) {
    test.skip(true, 'integration: could not apply customer scope');
    return;
  }

  const campaignId = await fetchFirstCampaignId(page);
  if (!campaignId) {
    test.skip(true, 'integration: no campaigns for platform sync');
    return;
  }

  await page.locator('#platform-campaign-id').fill(campaignId);

  const syncPost = page.waitForResponse(
    (response) =>
      response.request().method() === 'POST' &&
      response.url().includes('/api/v1/platform-campaigns/sync-run') &&
      response.status() === 204,
    { timeout: 20_000 },
  );
  const refreshedLinks = page.waitForResponse(isApiGet('/api/v1/platform-campaigns/links'), {
    timeout: 20_000,
  });

  await page.getByRole('button', { name: 'Run platform sync', exact: true }).click();

  await syncPost;
  await refreshedLinks;

  await expect(
    page.getByText('Platform sync completed for campaign.', { exact: true }),
  ).toBeVisible({ timeout: 15_000 });
});

test('postback DLQ retry posts to API and refreshes snapshot', async ({ page }) => {
  await loginAsAdmin(page);

  const snapshotGet = page.waitForResponse(isApiGet('/api/v1/postbacks/snapshot'), {
    timeout: 20_000,
  });
  await gotoLive(page, '/integrations/postbacks');
  await snapshotGet;

  const dlqResponse = await page.request.get(
    new URL('/api/v1/postbacks/snapshot', baseURL).toString(),
  );
  if (!dlqResponse.ok()) {
    test.skip(true, 'integration: postbacks snapshot unavailable');
    return;
  }

  const dlqBody = await dlqResponse.json();
  const dlqRows = Array.isArray(dlqBody.dlq) ? dlqBody.dlq : [];
  if (dlqRows.length === 0) {
    test.skip(true, 'integration: postback DLQ empty');
    return;
  }

  const dlqId = String(dlqRows[0].id ?? '');
  if (!dlqId) {
    test.skip(true, 'integration: postback DLQ row missing id');
    return;
  }

  await page.getByRole('button', { name: 'DLQ', exact: true }).click();
  await expect(page.getByRole('cell', { name: dlqId, exact: true })).toBeVisible({
    timeout: 15_000,
  });

  const retryPost = page.waitForResponse(
    (response) =>
      response.request().method() === 'POST' &&
      response.url().includes(`/api/v1/postbacks/dlq/${dlqId}/retry`) &&
      response.status() === 200,
    { timeout: 20_000 },
  );
  const refreshedSnapshot = page.waitForResponse(isApiGet('/api/v1/postbacks/snapshot'), {
    timeout: 20_000,
  });

  await page.getByRole('button', { name: 'Retry', exact: true }).first().click();

  const retryResponse = await retryPost;
  const retryBody = await retryResponse.json();
  expect(retryBody.status).toBe('ok');

  await refreshedSnapshot;
});
