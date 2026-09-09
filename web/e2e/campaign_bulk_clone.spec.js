import { test, expect } from '@playwright/test';

import {
  apiMutationHeaders,
  baseURL,
  fetchFirstCloneableCampaignId,
  integrationRunToken,
  loginAsAdmin,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('POST /api/v1/campaigns/bulk-clone clones selected campaign', async ({ page }) => {
  await loginAsAdmin(page);

  const campaignId = await fetchFirstCloneableCampaignId(page);
  if (!campaignId) {
    test.skip(true, 'integration: no cloneable campaign (run db seed-clone-balance)');
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
      headers: await apiMutationHeaders(page, {
        'Idempotency-Key': `bulk-clone-e2e-${integrationRunToken()}`,
      }),
    }
  );

  if (cloneResponse.status() === 405) {
    test.skip(
      true,
      'integration: control image missing POST /api/v1/campaigns/bulk-clone; rebuild control'
    );
    return;
  }

  expect(cloneResponse.ok()).toBeTruthy();
  const body = await cloneResponse.json();
  expect(Array.isArray(body.results)).toBe(true);
  expect(body.results).toHaveLength(1);
  expect(body.results[0].ok).toBe(true);
  expect(body.results[0].id).toBeTruthy();
  expect(body.results[0].name).toContain('(e2e bulk)');
});
