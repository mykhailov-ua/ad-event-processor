import { test, expect } from '@playwright/test';

import {
  fetchSessionCustomerId,
  gotoLive,
  loginAsAdmin,
  mainContent,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test(
  'click log report shows export stub without table runner GET',
  { tag: '@L1' },
  async ({ page }) => {
    await loginAsAdmin(page);
    const customerId = await fetchSessionCustomerId(page);
    if (!customerId) {
      test.skip(true, 'integration: click-log export stub requires session customer_id');
      return;
    }

    let clickLogListRequested = false;
    await page.route('**/api/v1/reports/click-log**', async (route) => {
      if (route.request().method() === 'GET') {
        clickLogListRequested = true;
      }
      await route.continue();
    });

    await gotoLive(page, `/reports/click-log?customer_id=${encodeURIComponent(customerId)}`);

    const main = mainContent(page);
    await expect(main.getByText('Control Plane export mode', { exact: true })).toBeVisible({
      timeout: 15_000,
    });

    const exportHubLink = main.getByRole('link', { name: 'Open Export Hub' });
    await expect(exportHubLink).toBeVisible();
    const href = await exportHubLink.getAttribute('href');
    expect(href).toMatch(/report_key=click-log/);

    expect(clickLogListRequested).toBe(false);
  }
);
