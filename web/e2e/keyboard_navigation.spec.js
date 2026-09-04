import { test, expect } from '@playwright/test';

import { gotoCampaigns, gotoCustomers, loginAsAdmin, skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('skip link moves focus to main content', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/campaigns');

  await page.keyboard.press('Tab');
  const skipLink = page.getByRole('link', { name: 'Skip to content' });
  await expect(skipLink).toBeFocused();

  await page.keyboard.press('Enter');
  await expect(page.locator('#main-content')).toBeFocused();
});

test('campaigns directory exposes keyboard-reachable filters', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCampaigns(page);

  await page.getByLabel('Search').focus();
  await expect(page.getByLabel('Search')).toBeFocused();

  await page.getByLabel('Period').focus();
  await expect(page.getByLabel('Period')).toBeFocused();
});

test('customers directory exposes keyboard-reachable table sort', async ({ page }) => {
  await loginAsAdmin(page);
  await gotoCustomers(page);

  const nameSort = page.getByRole('button', { name: 'Name' });
  await expect(nameSort).toBeVisible({ timeout: 15_000 });
  await nameSort.focus();
  await expect(nameSort).toBeFocused();
});

test('team page exposes keyboard-reachable invite control', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/team');

  const inviteButton = page.getByRole('button', { name: 'Invite member' });
  await inviteButton.focus();
  await expect(inviteButton).toBeFocused();
});

test('buyer dashboard chart legend toggles with keyboard', async ({ page }) => {
  await loginAsAdmin(page);
  await page.goto('/dashboards/buyer');
  await page.getByRole('button', { name: /^apply$/i }).click();

  const clicksLegend = page.getByRole('button', { name: 'Clicks', pressed: true });
  await expect(clicksLegend).toBeVisible({ timeout: 30_000 });

  await clicksLegend.focus();
  await expect(clicksLegend).toBeFocused();

  await page.keyboard.press('Space');
  await expect(page.getByRole('button', { name: 'Clicks', pressed: false })).toBeVisible();
});
