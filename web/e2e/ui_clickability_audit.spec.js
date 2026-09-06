import { test, expect } from '@playwright/test';
import { writeFileSync, mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

import {
  CLICKABILITY_AUDIT_ROUTES,
  auditButtonsOnPage,
  formatClickabilityReport,
} from './clickability_audit.js';
import { loginAsAdmin } from './helpers.js';

async function probeControlHealth() {
  const controlUrl =
    process.env.CONTROL_URL || process.env.ADMIN_CONTROL_URL || 'http://127.0.0.1:8188';
  const metaUrl = new URL('/api/v1/meta', controlUrl).toString();
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 5000);
  try {
    const response = await fetch(metaUrl, { signal: controller.signal });
    return response.ok;
  } catch {
    return false;
  } finally {
    clearTimeout(timeout);
  }
}

test.describe.configure({ mode: 'serial' });

test.beforeEach(async ({}, testInfo) => {
  if (process.env.ADMIN_E2E_SKIP === '1') {
    testInfo.skip(true, 'integration: ADMIN_E2E_SKIP=1');
    return;
  }
  if (!(await probeControlHealth())) {
    testInfo.skip(true, 'integration: control plane unreachable at :8188');
  }
});

test('audit enabled buttons are clickable across admin routes', async ({ page }) => {
  test.setTimeout(600_000);

  await loginAsAdmin(page);

  const allIssues = [];

  for (const path of CLICKABILITY_AUDIT_ROUTES) {
    try {
      const issues = await auditButtonsOnPage(page, path);
      allIssues.push(...issues);
    } catch (err) {
      const detail = err instanceof Error ? err.message : String(err);
      allIssues.push({
        path,
        label: '*',
        kind: 'navigation',
        detail: detail.slice(0, 240),
      });
    }
  }

  const report = formatClickabilityReport(allIssues);
  const reportDir = join(dirname(fileURLToPath(import.meta.url)), 'test-results');
  mkdirSync(reportDir, { recursive: true });
  writeFileSync(join(reportDir, 'clickability-audit.txt'), `${report}\n`, 'utf8');
  writeFileSync(
    join(reportDir, 'clickability-audit.json'),
    JSON.stringify(allIssues, null, 2),
    'utf8'
  );

  // eslint-disable-next-line no-console
  console.log(`\n${report}\n`);

  expect(allIssues, `Non-clickable buttons found:\n${report}`).toEqual([]);
});
