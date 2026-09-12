import { test, expect } from '@playwright/test';

import {
  MEDIA_BUYER_SMOKE_PRIMARY_GET_AUDIT,
  baseURL,
  gotoLive,
  loginWithCredentials,
  mainContent,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

/**
 * @returns {{ email: string | undefined, password: string | undefined }}
 */
function mediaBuyerCredentials() {
  return {
    email: process.env.ADMIN_E2E_MB_EMAIL || process.env.ADMIN_E2E_MEDIA_BUYER_EMAIL,
    password: process.env.ADMIN_E2E_MB_PASSWORD || process.env.ADMIN_E2E_MEDIA_BUYER_PASSWORD,
  };
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ path: string, api: string, permissionSlug: string }} row
 */
async function expectMediaBuyerDeepLinkDenied(page, row) {
  await gotoLive(page, row.path);

  const content = mainContent(page);
  await expect(
    content.getByRole('alert').or(content.getByText('403', { exact: true })).first()
  ).toBeVisible({ timeout: 15_000 });
  await expect(content.getByText(row.permissionSlug, { exact: true })).toBeVisible();

  const response = await page.request.get(new URL(row.api, baseURL).toString());
  expect(response.status(), row.api).toBe(403);
}

test('media buyer smoke-matrix primary GET RBAC audit', { tag: '@L1' }, async ({ page }) => {
  const { email, password } = mediaBuyerCredentials();
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

test(
  'media buyer /ops deep link denied with UI and API proof',
  { tag: '@L3' },
  async ({ page }) => {
    const { email, password } = mediaBuyerCredentials();
    if (!email || !password) {
      test.skip(true, 'integration: set ADMIN_E2E_MB_EMAIL and ADMIN_E2E_MB_PASSWORD');
      return;
    }

    await loginWithCredentials(page, email, password);
    await expectMediaBuyerDeepLinkDenied(page, {
      path: '/ops',
      api: '/api/v1/ops/home',
      permissionSlug: 'shards:read',
    });
  }
);

test(
  'media buyer /audit deep link denied with UI and API proof',
  { tag: '@L3' },
  async ({ page }) => {
    const { email, password } = mediaBuyerCredentials();
    if (!email || !password) {
      test.skip(true, 'integration: set ADMIN_E2E_MB_EMAIL and ADMIN_E2E_MB_PASSWORD');
      return;
    }

    await loginWithCredentials(page, email, password);
    await expectMediaBuyerDeepLinkDenied(page, {
      path: '/audit',
      api: '/api/v1/audit',
      permissionSlug: 'audit:read',
    });
  }
);

test(
  'media buyer /settings deep link denied with UI and API proof',
  { tag: '@L3' },
  async ({ page }) => {
    const { email, password } = mediaBuyerCredentials();
    if (!email || !password) {
      test.skip(true, 'integration: set ADMIN_E2E_MB_EMAIL and ADMIN_E2E_MB_PASSWORD');
      return;
    }

    await loginWithCredentials(page, email, password);
    await expectMediaBuyerDeepLinkDenied(page, {
      path: '/settings',
      api: '/api/v1/settings/platform',
      permissionSlug: 'settings:read',
    });
  }
);
