import { test, expect } from '@playwright/test';

import {
  fetchSessionCustomerId,
  gotoLiveTeam,
  integrationRunToken,
  integrationTeamInviteEmail,
  isApiGet,
  isApiPost,
  loginAsAdmin,
  mainHeading,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('team invite posts member and refreshes roster', async ({ page }) => {
  await loginAsAdmin(page);
  const runToken = integrationRunToken();
  const inviteEmail = integrationTeamInviteEmail(runToken);

  const customerId = await fetchSessionCustomerId(page);
  if (!customerId) {
    test.skip(true, 'integration: no default_customer_id in session');
    return;
  }

  const loaded = await gotoLiveTeam(page, customerId);
  if (!loaded) {
    test.skip(true, 'integration: team overview unavailable for session customer');
    return;
  }
  await expect(mainHeading(page, 'Team')).toBeVisible({ timeout: 15_000 });

  const membersLoaded = page.waitForResponse(
    (response) =>
      isApiGet('/api/v1/team/members')(response) &&
      response.url().includes(`customer_id=${encodeURIComponent(customerId)}`),
    { timeout: 20_000 },
  );
  await membersLoaded;

  await page.getByRole('button', { name: 'Invite member' }).click();
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.locator('#team-invite-email').fill(inviteEmail);
  await page.locator('#team-invite-role').fill('MB');

  const invitePost = page.waitForResponse(
    (response) =>
      isApiPost('/api/v1/team/members', 201)(response) &&
      response.url().includes(`customer_id=${encodeURIComponent(customerId)}`),
    { timeout: 20_000 },
  );
  const refreshedMembers = page.waitForResponse(isApiGet('/api/v1/team/members'), {
    timeout: 20_000,
  });

  await page.getByRole('button', { name: 'Send invite' }).click();

  const inviteResponse = await invitePost;
  const inviteBody = await inviteResponse.json();
  expect(inviteBody.email).toBe(inviteEmail);

  const membersResponse = await refreshedMembers;
  const membersBody = await membersResponse.json();
  expect(membersBody.items).toEqual(
    expect.arrayContaining([expect.objectContaining({ email: inviteEmail })]),
  );

  await expect(page.getByRole('cell', { name: inviteEmail, exact: true })).toBeVisible({
    timeout: 15_000,
  });
});
