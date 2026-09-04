/** @typedef {import('@playwright/test').Page} Page */
/** @typedef {import('@playwright/test').TestType} TestType */

export const baseURL =
  process.env.ADMIN_E2E_BASE_URL ||
  process.env.PLAYWRIGHT_BASE_URL ||
  'http://localhost:8188';

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
 * @param {Page} page
 */
export async function loginAsAdmin(page) {
  const { email, password } = getAdminCredentials();

  await page.goto('/login');
  await page.getByLabel('Email').fill(email);
  await page.getByLabel('Password').fill(password);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await page.waitForURL((url) => !url.pathname.endsWith('/login'), { timeout: 15_000 });
}

/**
 * @param {Page} page
 */
export async function gotoCustomers(page) {
  await page.goto('/customers');
  await page.getByRole('heading', { name: 'Customers' }).waitFor();
}

/**
 * @param {Page} page
 */
export async function gotoCampaigns(page) {
  await page.goto('/campaigns');
  await page.getByRole('heading', { name: 'Campaigns' }).waitFor({ timeout: 15_000 });
  await page.getByRole('button', { name: 'Create', exact: true }).waitFor({ timeout: 15_000 });
}

/**
 * @param {Page} page
 */
export async function gotoBilling(page) {
  await page.goto('/billing');
  await page.getByRole('heading', { name: 'Billing' }).waitFor();
}

/**
 * @param {Page} page
 */
export async function gotoOps(page) {
  await page.goto('/ops');
}
