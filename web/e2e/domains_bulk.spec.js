import { test, expect } from '@playwright/test';

import {
  baseURL,
  integrationRunToken,
  loginAsAdmin,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('POST /api/v1/ops/domains/bulk accepts CSV hostnames and returns 202 job', async ({
  page,
}) => {
  await loginAsAdmin(page);

  const token = integrationRunToken();
  const response = await page.request.post(
    new URL('/api/v1/ops/domains/bulk', baseURL).toString(),
    {
      data: {
        csv: `hostname\nbulk-e2e-a-${token}.test\nbulk-e2e-b-${token}.test`,
        cloudflare_zone_id: 'zone-e2e-bulk',
      },
    }
  );

  expect(response.status()).toBe(202);
  const body = await response.json();
  expect(body.job_id).toBeTruthy();
  expect(body.total).toBe(2);
  expect(body.status).toBe('pending');
});
