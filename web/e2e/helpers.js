/** @typedef {import('@playwright/test').Page} Page */
/** @typedef {import('@playwright/test').TestType} TestType */

import { randomBytes } from 'node:crypto';

import { expect } from '@playwright/test';

export const baseURL =
  process.env.ADMIN_E2E_BASE_URL || process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:8188';

const DEFAULT_EMAIL = 'admin@test.local';
const DEFAULT_PASSWORD = 'Password123!';

/**
 * @returns {{ email: string, password: string }}
 */
export function getAdminCredentials() {
  const email =
    process.env.ADMIN_E2E_USER ||
    process.env.ADMIN_E2E_EMAIL ||
    process.env.ADMIN_STACK_E2E_EMAIL ||
    process.env.ADMIN_BOOTSTRAP_EMAIL ||
    DEFAULT_EMAIL;
  const password =
    process.env.ADMIN_E2E_PASSWORD ||
    process.env.ADMIN_STACK_E2E_PASSWORD ||
    process.env.ADMIN_BOOTSTRAP_PASSWORD ||
    DEFAULT_PASSWORD;
  return { email, password };
}

/**
 * @param {import('@playwright/test').TestInfo} testInfo
 */
export async function skipUnlessIntegrationReady(testInfo) {
  if (process.env.ADMIN_E2E_SKIP === '1') {
    testInfo.skip(true, 'integration: ADMIN_E2E_SKIP=1');
    return;
  }

  const reachable = await probeBaseUrl();
  if (!reachable) {
    testInfo.skip(true, `integration: control plane unreachable at ${baseURL}`);
  }
}

/**
 * @returns {Promise<boolean>}
 */
export async function probeBaseUrl() {
  const healthUrl = new URL('/health', baseURL).toString();
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 5000);

  try {
    const response = await fetch(healthUrl, { signal: controller.signal });
    return response.ok;
  } catch {
    return false;
  } finally {
    clearTimeout(timeout);
  }
}

/**
 * @param {import('@playwright/test').Page} page
 */
export async function assertLiveApiMode(_page) {
  // No-op: in-browser dev_mock tier removed; E2E defaults to live control (:8188).
}

/**
 * Authenticated page body (excludes breadcrumb header chrome).
 * @param {import('@playwright/test').Page} page
 */
export function mainContent(page) {
  return page.locator('#main-content');
}

/**
 * Page title inside #main-content (not breadcrumb duplicate headings).
 * @param {import('@playwright/test').Page} page
 * @param {string} name
 */
export function mainHeading(page, name) {
  return mainContent(page).getByRole('heading', { name, exact: true });
}

/**
 * Standalone auth pages render outside the app shell (no #main-content).
 * @param {import('@playwright/test').Page} page
 */
export function loginSignInHeading(page) {
  return page.getByRole('heading', { name: 'Sign in', exact: true });
}

/**
 * Primary sidebar navigation landmark.
 * @param {import('@playwright/test').Page} page
 */
export function mainNav(page) {
  return page.getByRole('navigation', { name: 'Main' });
}

/**
 * Hub bento card link inside page content.
 * @param {import('@playwright/test').Page} page
 * @param {string} href
 */
export function hubCardLink(page, href) {
  return mainContent(page).locator(`a[href="${href}"]`);
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} title
 */
export async function expandDetailsSection(page, title) {
  const section = page.locator('details').filter({
    has: page.getByText(title, { exact: true }),
  });
  const isOpen = await section.evaluate((element) => element.open);
  if (!isOpen) {
    await section.locator('summary').click();
  }
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} path
 */
export async function gotoLive(page, path) {
  const normalized = path.startsWith('/') ? path : `/${path}`;
  await page.goto(normalized);
  await assertLiveApiMode(page);
}

/**
 * @param {string} pathPart
 * @param {number} [status=200]
 */
export function isApiGet(pathPart, status = 200) {
  return (response) =>
    response.request().method() === 'GET' &&
    response.url().includes(pathPart) &&
    response.status() === status;
}

/**
 * @param {string} pathPart
 * @param {number} [status=200]
 */
export function isApiPost(pathPart, status = 200) {
  return (response) =>
    response.request().method() === 'POST' &&
    response.url().includes(pathPart) &&
    response.status() === status;
}

/**
 * @param {string} pathPart
 * @param {number} [status=200]
 */
export function isApiPatch(pathPart, status = 200) {
  return (response) =>
    response.request().method() === 'PATCH' &&
    response.url().includes(pathPart) &&
    response.status() === status;
}

/**
 * Navigate and wait for a GET API response.
 * @param {import('@playwright/test').Page} page
 * @param {string} path
 * @param {string} apiPathPart
 * @param {number} [status=200]
 * @returns {Promise<import('@playwright/test').Response>}
 */
export async function gotoLiveAwaitGet(page, path, apiPathPart, status = 200) {
  const listResponse = page.waitForResponse(isApiGet(apiPathPart, status), { timeout: 20_000 });
  await gotoLive(page, path);
  return listResponse;
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {(response: import('@playwright/test').Response) => boolean} predicate
 * @returns {Promise<import('@playwright/test').Response>}
 */
export async function gotoLiveAwaitResponse(page, path, predicate) {
  const listResponse = page.waitForResponse(predicate, { timeout: 20_000 });
  await gotoLive(page, path);
  return listResponse;
}

/** L1 smoke matrix routes with primary read GET (hub-only routes omit api). */
export const ADMIN_SMOKE_ROUTE_READS = [
  { path: '/customers', heading: 'Customers', api: '/api/v1/customers' },
  { path: '/campaigns', heading: 'Campaigns', useCampaignsList: true },
  { path: '/billing', heading: 'Billing', api: '/api/v1/billing/invoices' },
  { path: '/settings', heading: 'Platform settings', api: '/api/v1/settings/platform' },
  { path: '/settings/license', heading: 'License', api: '/api/v1/license/status' },
  { path: '/team', heading: 'Team', api: '/api/v1/team/overview', skipWithoutCustomer: true },
  { path: '/audit', heading: 'Audit', api: '/api/v1/audit' },
  { path: '/reports', heading: 'Reports', api: '/api/v1/reports/catalog' },
  { path: '/ops', heading: 'Ops', api: '/api/v1/ops/home' },
  { path: '/fraud/presets', heading: 'Fraud presets', api: '/api/v1/fraud/presets' },
];

/** Ops section pages with list/read GET on mount. */
export const OPS_SECTION_READS = [
  { path: '/ops/dlq', api: '/api/v1/ops/dlq/inbox' },
  { path: '/ops/blacklist', api: '/api/v1/ops/blacklist' },
  { path: '/ops/incidents', api: '/api/v1/ops/incidents' },
  { path: '/ops/outbox', api: '/api/v1/ops/outbox' },
  { path: '/ops/shards', api: '/api/v1/ops/shards' },
  { path: '/ops/ml-model', api: '/api/v1/ops/ml-model' },
  { path: '/ops/domains', api: '/api/v1/ops/domains/rotation' },
  { path: '/ops/recon', api: '/api/v1/ops/recon' },
  { path: '/ops/consent', api: '/api/v1/ops/consent/proofs' },
  { path: '/ops/rum', api: '/api/v1/ops/rum' },
  { path: '/ops/metrics', api: '/api/v1/ops/dashboard/metrics' },
];

/**
 * @returns {string}
 */
export function randomHex32() {
  return randomBytes(16).toString('hex');
}

/**
 * Unique token for integration fixtures (timestamp; safe in emails and DNS labels).
 * @returns {string}
 */
export function integrationRunToken() {
  return String(Date.now());
}

/**
 * @param {string} [runToken]
 * @returns {string}
 */
export function integrationTeamInviteEmail(runToken = integrationRunToken()) {
  return `team.invite.${runToken}@test.local`;
}

/**
 * @param {string} [runToken]
 * @returns {string}
 */
export function integrationSettingsTrackingDomain(runToken = integrationRunToken()) {
  return `settings.patch.probe.${runToken}.invalid`;
}

/**
 * @param {string} [runToken]
 * @returns {string}
 */
export function integrationFraudLabelReason(runToken = integrationRunToken()) {
  return `manual fraud label integration ${runToken}`;
}

/**
 * @param {string} [runToken]
 * @returns {string}
 */
export function integrationCampaignValidateSuffix(runToken = integrationRunToken()) {
  return ` validate probe ${runToken}`;
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<string>}
 */
export async function fetchSessionCustomerId(page) {
  const response = await page.request.get(new URL('/api/v1/session', baseURL).toString());
  if (!response.ok()) {
    return '';
  }
  const body = await response.json();
  const customerId = body?.default_customer_id;
  return typeof customerId === 'string' ? customerId.trim() : '';
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {{ inputSelector: string, buttonName: string, listPathPart: string }} options
 * @returns {Promise<boolean>}
 */
export async function ensureCustomerScopeLoaded(page, options) {
  const { inputSelector, buttonName, listPathPart } = options;
  const customerId = await fetchSessionCustomerId(page);
  if (!customerId) {
    return false;
  }

  const input = page.locator(inputSelector);
  const current = (await input.inputValue().catch(() => '')).trim();
  if (current !== customerId) {
    await input.fill(customerId);
  }

  const listResponse = page.waitForResponse(isApiGet(listPathPart), { timeout: 20_000 });
  await page.getByRole('button', { name: buttonName, exact: true }).click();
  await listResponse;
  return true;
}

/**
 * @param {import('@playwright/test').Page} page
 */
export async function applyCustomerScopeIfPrompted(page) {
  const customerRequired = page.getByText('Customer required', { exact: true });
  if (await customerRequired.isVisible({ timeout: 3000 }).catch(() => false)) {
    await page.getByRole('button', { name: 'Apply' }).click();
  }
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {unknown} body
 * @param {{ emptyTitle: string, rowLabel: (row: Record<string, unknown>) => string }} options
 */
export async function expectApiListBoundToDom(page, body, options) {
  const { emptyTitle, rowLabel } = options;
  const items = Array.isArray(body) ? body : body.items;
  expect(Array.isArray(items)).toBe(true);
  if (!Array.isArray(body) && body != null && typeof body === 'object' && 'total' in body) {
    expect(typeof body.total).toBe('number');
  }

  if (items.length === 0) {
    await expect(page.getByText(emptyTitle, { exact: true })).toBeVisible({ timeout: 15_000 });
    return;
  }

  const main = mainContent(page);
  for (const item of items) {
    const label = rowLabel(item);
    expect(typeof label).toBe('string');
    expect(label.length).toBeGreaterThan(0);

    // FE2 (scoped): .or() disambiguates row label across cell/link/input/text surfaces only.
    // Do not chain .or(empty) or stub headings on data tables (L1s ban; see web/e2e/README.md).
    const rowLocator = main
      .getByRole('cell', { name: label, exact: true })
      .or(main.getByRole('link', { name: label, exact: true }))
      .or(main.locator(`input[value="${label}"]`))
      .or(main.getByText(label, { exact: true }));
    if (
      await rowLocator
        .first()
        .isVisible({ timeout: 3000 })
        .catch(() => false)
    ) {
      return;
    }
  }

  const firstLabel = rowLabel(items[0]);
  const fallbackLocator = main
    .getByRole('cell', { name: firstLabel, exact: true })
    .or(main.getByRole('link', { name: firstLabel, exact: true }))
    .or(main.locator(`input[value="${firstLabel}"]`))
    .or(main.getByText(firstLabel, { exact: true }));
  await expect(fallbackLocator.first()).toBeVisible({ timeout: 15_000 });
}

/**
 * @param {import('@playwright/test').Response} response
 */
export function isCampaignsListResponse(response) {
  const url = response.url();
  return (
    response.request().method() === 'GET' &&
    url.includes('/api/v1/campaigns') &&
    !url.includes('/metrics') &&
    !url.includes('/list-facets') &&
    !url.includes('/export') &&
    !url.includes('/onboarding-templates') &&
    !url.includes('/wizard/') &&
    response.status() === 200
  );
}

/**
 * @param {import('@playwright/test').Page} page
 */
export async function gotoCampaignsLive(page) {
  await page.goto('/campaigns');
  await mainHeading(page, 'Campaigns').waitFor({ timeout: 15_000 });
  await assertLiveApiMode(page);
}

/**
 * @param {Page} page
 */
export async function loginAsAdmin(page) {
  const { email, password } = getAdminCredentials();

  for (let attempt = 0; attempt < 2; attempt++) {
    await page.goto('/login');
    await page.getByLabel('Email').fill(email);
    await page.getByLabel('Password').fill(password);
    await page.getByRole('button', { name: 'Sign in' }).click();

    const navigated = await page
      .waitForURL((url) => !url.pathname.endsWith('/login'), { timeout: 15_000 })
      .then(() => true)
      .catch(() => false);
    if (navigated) {
      await mainNav(page).waitFor({ timeout: 15_000 });
      return;
    }

    if (attempt === 1) {
      const invalidCredentials = await page
        .getByText('invalid credentials')
        .isVisible()
        .catch(() => false);
      throw new Error(
        invalidCredentials ? 'login failed: invalid credentials' : 'login failed: still on /login'
      );
    }
  }
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} [customerId]
 */
export async function gotoLiveTeam(page, customerId = '') {
  const scopedCustomerId = customerId || (await fetchSessionCustomerId(page));
  const path = scopedCustomerId
    ? `/team?customer_id=${encodeURIComponent(scopedCustomerId)}`
    : '/team';
  await gotoLive(page, path);
  const teamError = page.getByText('Could not load team overview', { exact: true });
  if (await teamError.isVisible({ timeout: 3000 }).catch(() => false)) {
    return false;
  }
  await mainHeading(page, 'Team').waitFor({ timeout: 15_000 });
  return true;
}

/**
 * @param {Page} page
 */
export async function gotoCustomers(page) {
  await page.goto('/customers');
  await mainHeading(page, 'Customers').waitFor();
}

/**
 * @param {Page} page
 */
export async function gotoCampaigns(page) {
  await page.goto('/campaigns');
  await mainHeading(page, 'Campaigns').waitFor({ timeout: 15_000 });
  await page
    .getByRole('button', { name: 'Quick create', exact: true })
    .waitFor({ timeout: 15_000 });
}

/**
 * @param {Page} page
 */
export async function gotoBilling(page) {
  await page.goto('/billing');
  await mainHeading(page, 'Billing').waitFor();
}

/**
 * @param {Page} page
 */
export async function gotoOps(page) {
  await page.goto('/ops');
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<string | null>}
 */
export async function openFirstCampaignEditor(page) {
  const listResponse = page.waitForResponse(isCampaignsListResponse, { timeout: 20_000 });
  await gotoCampaignsLive(page);
  await listResponse;

  const editLink = page.locator('main a[href$="/edit"]').first();
  if ((await editLink.count()) === 0) {
    return null;
  }

  await editLink.click();
  await page.getByRole('heading', { name: 'Campaign settings' }).waitFor({ timeout: 15_000 });

  const match = page.url().match(/\/campaigns\/([^/]+)\/edit/);
  return match?.[1] ?? null;
}

/**
 * @param {import('@playwright/test').Page} page
 * @returns {Promise<string>}
 */
export async function fetchFirstCampaignId(page) {
  const response = await page.request.get(new URL('/api/v1/campaigns?limit=1', baseURL).toString());
  if (!response.ok()) {
    return '';
  }
  const body = await response.json();
  const first = body?.items?.[0];
  const id = first?.id;
  return typeof id === 'string' ? id : '';
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} listPathPart
 * @returns {Promise<boolean>}
 */
export async function ensureIntegrationCustomerScope(page, listPathPart) {
  return ensureCustomerScopeLoaded(page, {
    inputSelector: '#customer-scope-id',
    buttonName: 'Apply',
    listPathPart,
  });
}
