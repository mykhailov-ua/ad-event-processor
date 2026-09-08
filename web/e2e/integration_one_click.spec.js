import { test, expect } from '@playwright/test';

import {
  baseURL,
  fetchFirstCampaignId,
  loginAsAdmin,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('apply-templates dry-run returns postback URL template', async ({ page }) => {
  await loginAsAdmin(page);

  const campaignId = await fetchFirstCampaignId(page);
  if (!campaignId) {
    test.skip(true, 'integration: no campaigns available for apply-templates dry-run');
    return;
  }

  const dryRunResponse = await page.request.post(
    new URL(`/api/v1/campaigns/${campaignId}/apply-templates/dry-run`, baseURL).toString(),
    {
      data: {
        affiliate_network: 'affiliate_everad',
        tracking_domain: 'trk.example.com',
      },
    }
  );

  expect(dryRunResponse.ok()).toBeTruthy();
  const body = await dryRunResponse.json();
  expect(body.postback_url_template).toBeTruthy();
  expect(String(body.postback_url_template)).toContain('{');
});
