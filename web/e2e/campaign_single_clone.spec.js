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

test('POST /api/v1/campaigns/{id}/clone clones one campaign', async ({ page }) => {
  await loginAsAdmin(page);

  const campaignId = await fetchFirstCloneableCampaignId(page);
  if (!campaignId) {
    test.skip(true, 'integration: no cloneable campaign (run db seed-clone-balance)');
    return;
  }

  const cloneResponse = await page.request.post(
    new URL(`/api/v1/campaigns/${campaignId}/clone`, baseURL).toString(),
    {
      data: {
        name_suffix: ' (e2e single)',
      },
      headers: await apiMutationHeaders(page, {
        'Idempotency-Key': `single-clone-e2e-${integrationRunToken()}`,
      }),
    }
  );

  expect(cloneResponse.ok()).toBeTruthy();
  const body = await cloneResponse.json();
  expect(body.id).toBeTruthy();
  expect(body.name).toContain('(e2e single)');
});
