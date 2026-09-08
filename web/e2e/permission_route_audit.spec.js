import { test, expect } from '@playwright/test';

import {
  MEDIA_BUYER_SMOKE_PRIMARY_GET_AUDIT,
  baseURL,
  loginWithCredentials,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('media buyer smoke-matrix primary GET RBAC audit', { tag: '@L1' }, async ({ page }) => {
  const email = process.env.ADMIN_E2E_MB_EMAIL || process.env.ADMIN_E2E_MEDIA_BUYER_EMAIL;
  const password = process.env.ADMIN_E2E_MB_PASSWORD || process.env.ADMIN_E2E_MEDIA_BUYER_PASSWORD;
  if (!email || !password) {
    test.skip(true, 'integration: set ADMIN_E2E_MB_EMAIL and ADMIN_E2E_MB_PASSWORD');
    return;
  }

  await loginWithCredentials(page, email, password);

  for (const row of MEDIA_BUYER_SMOKE_PRIMARY_GET_AUDIT) {
    const response = await page.request.get(new URL(row.api, baseURL).toString());
    if (row.wantForbidden) {
      expect(response.status(), row.api).toBe(403);
      continue;
    }
    expect(response.status(), row.api).not.toBe(403);
  }
});
