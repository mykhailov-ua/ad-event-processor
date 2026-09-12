import { test, expect } from '@playwright/test';

import {
  apiMutationHeaders,
  baseURL,
  loginAsAdmin,
  skipUnlessIntegrationReady,
} from './helpers.js';

test.beforeEach(async ({}, testInfo) => {
  await skipUnlessIntegrationReady(testInfo);
});

test('hosted lander publish serves GET /lp/{id}/', { tag: '@write' }, async ({ page }) => {
  await loginAsAdmin(page);

  const landersResponse = await page.request.get(
    new URL('/api/v1/landers?limit=20&hosting=hosted', baseURL).toString()
  );
  if (!landersResponse.ok()) {
    test.skip(true, `integration: landers list ${landersResponse.status()}`);
    return;
  }
  const landersBody = await landersResponse.json();
  const landers = Array.isArray(landersBody) ? landersBody : (landersBody.items ?? []);
  const lander = landers.find((row) => row.hosting === 'hosted') ?? landers[0];
  if (!lander?.id) {
    test.skip(true, 'integration: no hosted lander available');
    return;
  }

  const editorStateResponse = await page.request.get(
    new URL(`/api/v1/landers/${lander.id}/hosted-editor`, baseURL).toString()
  );
  if (!editorStateResponse.ok()) {
    test.skip(true, `integration: hosted editor ${editorStateResponse.status()}`);
    return;
  }
  const editorState = await editorStateResponse.json();

  const html = `<!DOCTYPE html><html><head><title>parity</title></head><body><h1>hosted e2e</h1></body></html>`;
  const saveResponse = await page.request.put(
    new URL(`/api/v1/landers/${lander.id}/hosted-files/index.html`, baseURL).toString(),
    {
      headers: apiMutationHeaders(),
      data: { content: html },
    }
  );
  expect(saveResponse.ok()).toBeTruthy();

  const publishResponse = await page.request.post(
    new URL(`/api/v1/landers/${lander.id}/hosted-publish`, baseURL).toString(),
    {
      headers: apiMutationHeaders(),
      data: { version: editorState.draft_version },
    }
  );
  expect(publishResponse.ok()).toBeTruthy();

  const liveResponse = await page.request.get(new URL(`/lp/${lander.id}/`, baseURL).toString());
  expect(liveResponse.status()).toBe(200);
  const body = await liveResponse.text();
  expect(body).toContain('hosted e2e');
});
