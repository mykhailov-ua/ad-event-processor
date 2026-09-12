# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: login.spec.js >> login bad password shows ErrorBlock
- Location: login.spec.js:38:1

# Error details

```
Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:8188/login
Call log:
  - navigating to "http://localhost:8188/login", waiting until "load"

```

# Test source

```ts
  1  | import { test, expect } from '@playwright/test';
  2  | 
  3  | import {
  4  |   expectErrorBlockVisible,
  5  |   getAdminCredentials,
  6  |   loginAsAdmin,
  7  |   loginSignInHeading,
  8  |   openAppNavigation,
  9  |   skipUnlessIntegrationReady,
  10 |   stubApiRoute,
  11 | } from './helpers.js';
  12 | 
  13 | test.beforeEach(async ({}, testInfo) => {
  14 |   await skipUnlessIntegrationReady(testInfo);
  15 | });
  16 | 
  17 | test('login page loads sign-in form', async ({ page }) => {
  18 |   await page.goto('/login');
  19 | 
  20 |   await expect(loginSignInHeading(page)).toBeVisible();
  21 |   await expect(page.getByLabel('Email')).toBeVisible();
  22 |   await expect(page.getByRole('textbox', { name: 'Password' })).toBeVisible();
  23 |   await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible();
  24 | });
  25 | 
  26 | test('can submit admin credentials', async ({ page }) => {
  27 |   const { email, password } = getAdminCredentials();
  28 | 
  29 |   await page.goto('/login');
  30 |   await page.getByLabel('Email').fill(email);
  31 |   await page.getByRole('textbox', { name: 'Password' }).fill(password);
  32 |   await page.getByRole('button', { name: 'Sign in' }).click();
  33 | 
  34 |   await page.waitForURL((url) => !url.pathname.endsWith('/login'), { timeout: 15_000 });
  35 |   await expect(page).not.toHaveURL(/\/login$/);
  36 | });
  37 | 
  38 | test('login bad password shows ErrorBlock', { tag: '@L3' }, async ({ page }) => {
  39 |   await stubApiRoute(page, '/api/v1/auth/login', 401, {
  40 |     error: { code: 'UNAUTHORIZED', message: 'invalid credentials' },
  41 |   });
  42 | 
> 43 |   await page.goto('/login');
     |              ^ Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:8188/login
  44 |   await page.getByLabel('Email').fill('wrong@test.local');
  45 |   await page.getByRole('textbox', { name: 'Password' }).fill('WrongPassword123!');
  46 |   await page.getByRole('button', { name: 'Sign in' }).click();
  47 | 
  48 |   await expectErrorBlockVisible(page, 'Sign in failed');
  49 | });
  50 | 
  51 | test('loginAsAdmin helper reaches authenticated shell', async ({ page }) => {
  52 |   await loginAsAdmin(page);
  53 |   const nav = await openAppNavigation(page);
  54 |   const usersLink = nav.getByRole('link', { name: 'Users' });
  55 |   if (await usersLink.isVisible().catch(() => false)) {
  56 |     await expect(usersLink).toBeVisible();
  57 |     return;
  58 |   }
  59 |   await expect(page.getByRole('menuitem', { name: 'Users' })).toBeVisible();
  60 | });
  61 | 
```