import { test, expect } from '@playwright/test';

import { skipUnlessIntegrationReady } from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('setup page loads when install is pending', async ({ page }) => {
  await page.goto('/setup');

  const setupHeading = page.getByRole('heading', { name: 'Initial setup' });
  const signInHeading = page.getByRole('heading', { name: 'Sign in' });

  if (await setupHeading.isVisible({ timeout: 5000 }).catch(() => false)) {
    await expect(page.getByLabel('Setup token')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Complete setup' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Activate with license' })).toBeVisible();
    return;
  }

  await expect(signInHeading).toBeVisible();
});

test('activate page loads when install is pending', async ({ page }) => {
  await page.goto('/activate');

  const activateHeading = page.getByRole('heading', { name: 'Activate deployment' });
  const signInHeading = page.getByRole('heading', { name: 'Sign in' });

  if (await activateHeading.isVisible({ timeout: 5000 }).catch(() => false)) {
    await expect(page.getByLabel('License JWT')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Activate and sign in' })).toBeVisible();
    return;
  }

  await expect(signInHeading).toBeVisible();
});

test('login page links to setup paths', async ({ page }) => {
  await page.goto('/login');

  await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Run setup' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'activate with license' })).toBeVisible();
});
