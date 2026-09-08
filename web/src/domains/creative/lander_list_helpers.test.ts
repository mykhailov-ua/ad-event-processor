import assert from 'node:assert/strict';
import test from 'node:test';

import type { Lander } from '@/api/types';
import {
  landerEditorPath,
  landerHostedCellText,
  landerHostedCellTitle,
  resolveLanderHostingKind,
  resolveLanderLiveUrl,
  resolveLanderPrimaryUrl,
} from '@/domains/creative/lander_list_helpers';

const externalLander: Lander = {
  id: '00000000-0000-4000-8000-000000000001',
  name: 'External',
  url: 'https://example.com/lp',
  created_at: '2026-01-01T00:00:00.000Z',
};

const hostedLander: Lander = {
  id: '00000000-0000-4000-8000-000000000002',
  name: 'Hosted',
  hosted_asset_id: '00000000-0000-4000-8000-000000000099',
  hosted_url: 'https://trk.example/lp/00000000-0000-4000-8000-000000000002/',
  created_at: '2026-01-01T00:00:00.000Z',
};

test('resolveLanderHostingKind classifies external and hosted rows', () => {
  assert.equal(resolveLanderHostingKind(externalLander), 'external');
  assert.equal(resolveLanderHostingKind(hostedLander), 'hosted');
  assert.equal(
    resolveLanderHostingKind({
      ...externalLander,
      url: '',
    }),
    'unconfigured'
  );
});

test('resolveLanderPrimaryUrl prefers hosted_url', () => {
  assert.equal(resolveLanderPrimaryUrl(externalLander), 'https://example.com/lp');
  assert.equal(
    resolveLanderPrimaryUrl(hostedLander),
    'https://trk.example/lp/00000000-0000-4000-8000-000000000002/'
  );
});

test('resolveLanderLiveUrl builds index path for hosted landers', () => {
  assert.equal(resolveLanderLiveUrl(externalLander), undefined);
  assert.equal(
    resolveLanderLiveUrl(hostedLander),
    'https://trk.example/lp/00000000-0000-4000-8000-000000000002/index.html'
  );
});

test('landerEditorPath', () => {
  assert.equal(
    landerEditorPath('00000000-0000-4000-8000-000000000002'),
    '/landers/00000000-0000-4000-8000-000000000002/editor'
  );
});

test('landerHostedCellText flattens hosted state into one label', () => {
  assert.equal(landerHostedCellText(externalLander), 'External');
  assert.equal(
    landerHostedCellText({
      ...hostedLander,
      has_unpublished_draft: true,
      published_version: 3,
    }),
    'Hosted | Draft | v3'
  );
  assert.match(landerHostedCellTitle(hostedLander) ?? '', /Hosted/);
});
