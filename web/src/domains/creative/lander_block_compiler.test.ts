import assert from 'node:assert/strict';
import test from 'node:test';

import { compileLanderBlocksToHtml } from '@/domains/creative/lander_block_compiler';
import { newLanderBlock } from '@/domains/creative/lander_block_model';

test('compileLanderBlocksToHtml renders hero and cta blocks', () => {
  const html = compileLanderBlocksToHtml([
    { ...newLanderBlock('hero'), title: 'Welcome', subtitle: 'Offer inside' },
    { ...newLanderBlock('cta'), label: 'Go', href: 'https://trk.example.com/click' },
  ]);
  assert.match(html, /<h1>Welcome<\/h1>/);
  assert.match(html, /cta-button/);
  assert.match(html, /<!DOCTYPE html>/);
});
