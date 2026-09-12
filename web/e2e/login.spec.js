import { test, expect } from '@playwright/test';

import {
  expectErrorBlockVisible,
  getAdminCredentials,
  loginAsAdmin,
  loginSignInHeading,
  openAppNavigation,
  skipUnlessIntegrationReady,
  stubApiRoute,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('login page loads sign-in form', async ({ page }) => {
  await page.goto('/login');

  await expect(loginSignInHeading(page)).toBeVisible();
  await expect(page.getByLabel('Email')).toBeVisible();
  await expect(page.getByRole('textbox', { name: 'Password' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible();
});

test('can submit admin credentials', async ({ page }) => {
  const { email, password } = getAdminCredentials();

  await page.goto('/login');
  await page.getByLabel('Email').fill(email);
  await page.getByRole('textbox', { name: 'Password' }).fill(password);
  await page.getByRole('button', { name: 'Sign in' }).click();

  await page.waitForURL((url) => !url.pathname.endsWith('/login'), { timeout: 15_000 });
  await expect(page).not.toHaveURL(/\/login$/);
});

test('login bad password shows ErrorBlock', { tag: '@L3' }, async ({ page }) => {
  await stubApiRoute(page, '/api/v1/auth/login', 401, {
    error: { code: 'UNAUTHORIZED', message: 'invalid credentials' },
  });

  await page.goto('/login');
  await page.getByLabel('Email').fill('wrong@test.local');
  await page.getByRole('textbox', { name: 'Password' }).fill('WrongPassword123!');
  await page.getByRole('button', { name: 'Sign in' }).click();

  await expectErrorBlockVisible(page, 'Sign in failed');
});

test('loginAsAdmin helper reaches authenticated shell', async ({ page }) => {
  await loginAsAdmin(page);
  const nav = await openAppNavigation(page);
  const usersLink = nav.getByRole('link', { name: 'Users' });
  if (await usersLink.isVisible().catch(() => false)) {
    await expect(usersLink).toBeVisible();
    return;
  }
  await expect(page.getByRole('menuitem', { name: 'Users' })).toBeVisible();
});
