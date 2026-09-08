import { test, expect } from '@playwright/test';

import {
  baseURL,
  fetchFirstCampaignId,
  integrationRunToken,
  loginAsAdmin,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('POST /api/v1/campaigns/bulk-clone clones selected campaign', async ({ page }) => {
  await loginAsAdmin(page);

  const campaignId = await fetchFirstCampaignId(page);
  if (!campaignId) {
    test.skip(true, 'integration: no campaigns available for bulk clone');
    return;
  }

  const campaignResponse = await page.request.get(
    new URL(`/api/v1/campaigns/${campaignId}`, baseURL).toString()
  );
  expect(campaignResponse.ok()).toBeTruthy();
  const campaign = await campaignResponse.json();
  const customerId = campaign?.customer_id;
  if (!customerId) {
    test.skip(true, 'integration: campaign missing customer_id');
    return;
  }

  const cloneResponse = await page.request.post(
    new URL('/api/v1/campaigns/bulk-clone', baseURL).toString(),
    {
      data: {
        source_campaign_ids: [campaignId],
        customer_id: customerId,
        name_suffix: ' (e2e bulk)',
      },
      headers: {
        'Idempotency-Key': `bulk-clone-e2e-${integrationRunToken()}`,
      },
    }
  );

  expect(cloneResponse.ok()).toBeTruthy();
  const body = await cloneResponse.json();
  expect(Array.isArray(body.results)).toBe(true);
  expect(body.results).toHaveLength(1);
  expect(body.results[0].ok).toBe(true);
  expect(body.results[0].id).toBeTruthy();
  expect(body.results[0].name).toContain('(e2e bulk)');
});
