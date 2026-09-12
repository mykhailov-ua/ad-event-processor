# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: login.spec.js >> loginAsAdmin helper reaches authenticated shell
- Location: login.spec.js:51:1

# Error details

```
Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:8188/login
Call log:
  - navigating to "http://localhost:8188/login", waiting until "load"

```

# Test source

```ts
  1   | /** @typedef {import('@playwright/test').Page} Page */
  2   | /** @typedef {import('@playwright/test').TestType} TestType */
  3   | 
  4   | import { randomBytes } from 'node:crypto';
  5   | 
  6   | import { expect } from '@playwright/test';
  7   | 
  8   | export const baseURL =
  9   |   process.env.ADMIN_E2E_BASE_URL || process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:8188';
  10  | 
  11  | export const trackerBaseURL =
  12  |   process.env.ADMIN_E2E_TRACKER_URL || process.env.TRACKER_BASE_URL || 'http://127.0.0.1:8181';
  13  | 
  14  | /**
  15  |  * @returns {Promise<boolean>}
  16  |  */
  17  | export async function probeTrackerBaseUrl() {
  18  |   const clickProbe = new URL('/click?type=click', trackerBaseURL).toString();
  19  |   const controller = new AbortController();
  20  |   const timeout = setTimeout(() => controller.abort(), 5000);
  21  |   try {
  22  |     const response = await fetch(clickProbe, {
  23  |       method: 'GET',
  24  |       redirect: 'manual',
  25  |       signal: controller.signal,
  26  |     });
  27  |     return response.status >= 300 && response.status < 600;
  28  |   } catch {
  29  |     return false;
  30  |   } finally {
  31  |     clearTimeout(timeout);
  32  |   }
  33  | }
  34  | 
  35  | const DEFAULT_EMAIL = 'admin@test.local';
  36  | const DEFAULT_PASSWORD = 'Password123!';
  37  | 
  38  | /**
  39  |  * @returns {{ email: string, password: string }}
  40  |  */
  41  | export function getAdminCredentials() {
  42  |   const email =
  43  |     process.env.ADMIN_E2E_USER ||
  44  |     process.env.ADMIN_E2E_EMAIL ||
  45  |     process.env.ADMIN_STACK_E2E_EMAIL ||
  46  |     process.env.ADMIN_BOOTSTRAP_EMAIL ||
  47  |     DEFAULT_EMAIL;
  48  |   const password =
  49  |     process.env.ADMIN_E2E_PASSWORD ||
  50  |     process.env.ADMIN_STACK_E2E_PASSWORD ||
  51  |     process.env.ADMIN_BOOTSTRAP_PASSWORD ||
  52  |     DEFAULT_PASSWORD;
  53  |   return { email, password };
  54  | }
  55  | 
  56  | /**
  57  |  * @param {import('@playwright/test').Page} page
  58  |  * @param {string} email
  59  |  * @param {string} password
  60  |  */
  61  | export async function loginWithCredentials(page, email, password) {
  62  |   for (let attempt = 0; attempt < 2; attempt++) {
> 63  |     await page.goto('/login');
      |                ^ Error: page.goto: net::ERR_CONNECTION_REFUSED at http://localhost:8188/login
  64  |     await page.getByLabel('Email').fill(email);
  65  |     await page.getByRole('textbox', { name: 'Password' }).fill(password);
  66  |     await page.getByRole('button', { name: 'Sign in' }).click();
  67  | 
  68  |     const navigated = await page
  69  |       .waitForURL((url) => !url.pathname.endsWith('/login'), { timeout: 15_000 })
  70  |       .then(() => true)
  71  |       .catch(() => false);
  72  |     if (navigated) {
  73  |       return;
  74  |     }
  75  | 
  76  |     const loginError = page
  77  |       .getByRole('alert')
  78  |       .getByText(/Sign in failed|session expired|permission/i);
  79  |     const errorText = (await loginError.textContent().catch(() => null))?.trim();
  80  |     if (attempt === 1) {
  81  |       throw new Error(errorText ? `login failed: ${errorText}` : 'login failed: still on /login');
  82  |     }
  83  |     await page.waitForTimeout(500);
  84  |   }
  85  | }
  86  | 
  87  | /**
  88  |  * @param {import('@playwright/test').Page} page
  89  |  */
  90  | export async function ensureLoggedIn(page) {
  91  |   if (!page.url().includes('/login')) {
  92  |     const accountVisible = await page
  93  |       .getByRole('button', { name: 'Account' })
  94  |       .isVisible({ timeout: 1500 })
  95  |       .catch(() => false);
  96  |     if (accountVisible) {
  97  |       return;
  98  |     }
  99  |   }
  100 |   await loginAsAdmin(page);
  101 | }
  102 | 
  103 | /**
  104 |  * @param {import('@playwright/test').TestInfo} testInfo
  105 |  */
  106 | export async function skipUnlessIntegrationReady(testInfo) {
  107 |   if (process.env.ADMIN_E2E_SKIP === '1') {
  108 |     testInfo.skip(true, 'integration: ADMIN_E2E_SKIP=1');
  109 |     return;
  110 |   }
  111 | 
  112 |   const reachable = await probeBaseUrl();
  113 |   if (!reachable) {
  114 |     testInfo.skip(true, `integration: control plane unreachable at ${baseURL}`);
  115 |   }
  116 | }
  117 | 
  118 | /**
  119 |  * @returns {Promise<boolean>}
  120 |  */
  121 | export async function probeBaseUrl() {
  122 |   const healthUrl = new URL('/health', baseURL).toString();
  123 |   const controller = new AbortController();
  124 |   const timeout = setTimeout(() => controller.abort(), 5000);
  125 | 
  126 |   try {
  127 |     const response = await fetch(healthUrl, { signal: controller.signal });
  128 |     return response.ok;
  129 |   } catch {
  130 |     return false;
  131 |   } finally {
  132 |     clearTimeout(timeout);
  133 |   }
  134 | }
  135 | 
  136 | /**
  137 |  * @param {import('@playwright/test').Page} page
  138 |  */
  139 | export async function assertLiveApiMode(_page) {
  140 |   // No-op: in-browser dev_mock tier removed; E2E defaults to live control (:8188).
  141 | }
  142 | 
  143 | /**
  144 |  * Authenticated page body (excludes breadcrumb header chrome).
  145 |  * @param {import('@playwright/test').Page} page
  146 |  */
  147 | export function mainContent(page) {
  148 |   return page.locator('#main-content');
  149 | }
  150 | 
  151 | /**
  152 |  * Page title inside #main-content (not breadcrumb duplicate headings).
  153 |  * @param {import('@playwright/test').Page} page
  154 |  * @param {string} name
  155 |  */
  156 | export function mainHeading(page, name) {
  157 |   return mainContent(page).getByRole('heading', { name, exact: true });
  158 | }
  159 | 
  160 | /**
  161 |  * Standalone auth pages render outside the app shell (no #main-content).
  162 |  * @param {import('@playwright/test').Page} page
  163 |  */
```