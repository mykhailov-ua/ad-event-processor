import { test, expect } from '@playwright/test';
import { ADMIN_USER, installDialogAutoAccept, mockAuthedSession } from './helpers.js';

const CAMPAIGN = {
  id: 'camp-pb-1',
  name: 'Postback campaign',
  status: 'active',
  customer_id: 'cust-1',
};

test('postbacks tab loads config form', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/campaigns/camp-pb-1', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(CAMPAIGN),
      });
      return;
    }
    await route.continue();
  });

  await page.route('**/api/v1/postbacks/config/camp-pb-1**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        provider: 'meta',
        url_template: 'https://example.com/pb?cid={click_id}',
        test_event_code: 'TEST123',
      }),
    });
  });

  await page.goto('/campaigns/camp-pb-1?tab=postbacks');
  await expect(page.getByRole('tab', { name: 'Postbacks' })).toBeVisible();
  await expect(page.getByLabel('Provider')).toBeVisible();
  await expect(page.getByLabel('URL template')).toHaveValue(/example\.com/);
});

test('postbacks tab saves config via PUT', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/campaigns/camp-pb-1', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(CAMPAIGN),
      });
      return;
    }
    await route.continue();
  });

  let putBody = null;
  await page.route('**/api/v1/postbacks/config/camp-pb-1**', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ provider: 'meta', url_template: '' }),
      });
      return;
    }
    if (route.request().method() === 'PUT') {
      putBody = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(putBody),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/campaigns/camp-pb-1?tab=postbacks');
  await page.getByLabel('URL template').fill('https://track.example/pb');
  await page.getByRole('button', { name: 'Save postback config' }).click();
  expect(putBody?.url_template).toBe('https://track.example/pb');
});
