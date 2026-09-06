/** @typedef {import('@playwright/test').Page} Page */

import { ADMIN_SMOKE_ROUTE_READS, OPS_SECTION_READS, gotoLive } from './helpers.js';

/** Read-only routes for clickability sweep (no :id detail pages). */
export const CLICKABILITY_AUDIT_ROUTES = [
  '/dashboards/buyer',
  ...ADMIN_SMOKE_ROUTE_READS.map((route) => route.path),
  ...OPS_SECTION_READS.map((route) => route.path),
  '/fraud',
  '/fraud/presets',
  '/fraud/labels',
  '/fraud/overrides',
  '/fraud/integrations',
  '/fraud/decisions',
  '/integrations',
  '/integrations/cost-sync',
  '/integrations/postbacks',
  '/integrations/schemas',
  '/integrations/platform-campaigns',
  '/integrations/affiliate-presets',
  '/creative',
  '/flows',
  '/landers',
  '/offers',
  '/brands',
  '/domains',
  '/supply',
  '/supply/sellers',
  '/supply/ads-txt',
  '/automation',
  '/automation/rules',
  '/automation/presets',
  '/margin-guard/policies',
  '/margin-guard/activity',
  '/smart-alerts/rules',
  '/smart-alerts/history',
  '/traffic-optimizer/rules',
  '/traffic-optimizer/presets',
  '/rtb',
  '/rtb/deals',
  '/rtb/shadow',
  '/rtb/floors',
  '/rtb/validate',
  '/rtb/integration-profile',
  '/portals',
  '/selfserve',
  '/views',
  '/report-schedules',
  '/publisher/dashboard',
  '/publisher/statements',
  '/reports/jobs',
  '/reports/click-log',
  '/billing/exports',
  '/disputes',
  '/support/feedback',
  '/telegram/bots',
  '/telegram/postbacks',
];

/**
 * @typedef {{ path: string, label: string, kind: string, detail: string }} ClickabilityIssue
 */

/**
 * Scan visible enabled buttons in #main-content for obscured hit targets or non-actionable trial clicks.
 * @param {Page} page
 * @param {string} path
 * @returns {Promise<ClickabilityIssue[]>}
 */
export async function auditButtonsOnPage(page, path) {
  await gotoLive(page, path);
  await page.locator('#main-content').waitFor({ state: 'visible', timeout: 20_000 });
  await page.keyboard.press('Escape').catch(() => {});

  const buttons = page.locator('#main-content').getByRole('button');
  const count = await buttons.count();
  const issues = [];

  for (let index = 0; index < count; index += 1) {
    const button = buttons.nth(index);
    if (!(await button.isVisible().catch(() => false))) {
      continue;
    }

    const label = (await button.innerText().catch(() => ''))
      .trim()
      .replace(/\s+/g, ' ')
      .slice(0, 120);
    const ariaLabel = (await button.getAttribute('aria-label'))?.trim() ?? '';
    const resolvedLabel = label || ariaLabel || '[unlabeled]';

    if (!label && !ariaLabel) {
      issues.push({
        path,
        label: resolvedLabel,
        kind: 'missing-accessible-name',
        detail: 'button has no text or aria-label',
      });
      continue;
    }

    if (await button.isDisabled().catch(() => false)) {
      continue;
    }

    await button.scrollIntoViewIfNeeded().catch(() => {});

    const obscured = await button.evaluate((element) => {
      const rect = element.getBoundingClientRect();
      if (rect.width < 1 || rect.height < 1) {
        return 'zero-size';
      }
      const centerX = rect.left + rect.width / 2;
      const centerY = rect.top + rect.height / 2;
      const hit = document.elementFromPoint(centerX, centerY);
      if (!hit) {
        return 'no-element-at-point';
      }
      if (element === hit || element.contains(hit)) {
        return null;
      }
      const hitTag = hit.tagName.toLowerCase();
      const hitClass =
        hit.className && typeof hit.className === 'string'
          ? hit.className.split(/\s+/).slice(0, 2).join('.')
          : '';
      return `obscured-by-${hitTag}${hitClass ? `.${hitClass}` : ''}`;
    });

    if (obscured) {
      issues.push({ path, label: resolvedLabel, kind: 'obscured', detail: obscured });
      continue;
    }

    try {
      await button.click({ trial: true, timeout: 3000 });
    } catch (err) {
      const detail = err instanceof Error ? err.message : String(err);
      issues.push({
        path,
        label: resolvedLabel,
        kind: 'trial-click',
        detail: detail.slice(0, 240),
      });
    }
  }

  return issues;
}

/**
 * @param {ClickabilityIssue[]} issues
 * @returns {string}
 */
export function formatClickabilityReport(issues) {
  if (issues.length === 0) {
    return 'No non-clickable enabled buttons found.';
  }

  const lines = [`Found ${issues.length} clickability issue(s):`];
  for (const issue of issues) {
    lines.push(`- [${issue.kind}] ${issue.path} :: "${issue.label}" :: ${issue.detail}`);
  }
  return lines.join('\n');
}
