import { test, expect } from '@playwright/test';

import {
  getAdminCredentials,
  loginAsAdmin,
  loginSignInHeading,
  mainNav,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('login page loads sign-in form', async ({ page }) => {
  await page.goto('/login');

  await expect(loginSignInHeading(page)).toBeVisible();
  await expect(page.getByLabel('Email')).toBeVisible();
  await expect(page.getByLabel('Password')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible();
});

test('can submit admin credentials', async ({ page }) => {
  const { email, password } = getAdminCredentials();

  await page.goto('/login');
  await page.getByLabel('Email').fill(email);
  await page.getByLabel('Password').fill(password);
  await page.getByRole('button', { name: 'Sign in' }).click();

  await page.waitForURL((url) => !url.pathname.endsWith('/login'), { timeout: 15_000 });
  await expect(page).not.toHaveURL(/\/login$/);
});

test('loginAsAdmin helper reaches authenticated shell', async ({ page }) => {
  await loginAsAdmin(page);
  await expect(mainNav(page).getByRole('link', { name: 'Users' })).toBeVisible();
});
