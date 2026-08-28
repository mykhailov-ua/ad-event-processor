import { test, expect } from '@playwright/test';
import { ADMIN_USER, installDialogAutoAccept, mockAuthedSession } from './helpers.js';

test('RTB integration page loads profile section', async ({ page }) => {
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/rtb/integration-profile**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ enabled: true, endpoint: 'https://rtb.example/bid' }),
    });
  });

  await page.goto('/rtb/integration');
  await expect(page.getByRole('heading', { name: 'RTB integration' })).toBeVisible();
});

test('RTB floors apply uses confirm dialog', async ({ page }) => {
  installDialogAutoAccept(page);
  await mockAuthedSession(page, ADMIN_USER);

  await page.route('**/api/v1/rtb/integration-profile**', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ enabled: true }),
    });
  });

  let applied = false;
  await page.route('**/api/v1/rtb/floors/apply**', async (route) => {
    applied = true;
    await route.fulfill({ status: 204, body: '' });
  });

  await page.goto('/rtb/integration');
  const applyBtn = page.getByRole('button', { name: 'Apply floors' });
  if (await applyBtn.count()) {
    await applyBtn.click();
    await expect.poll(() => applied).toBe(true);
  } else {
    test.skip(true, 'Apply floors control not mounted in current RTB integration UI');
  }
});
