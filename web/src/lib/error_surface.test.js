import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { installErrorSurface } from './error_surface.js';

describe('error_surface', () => {
  /** @type {HTMLElement} */
  let root;

  beforeEach(() => {
    root = document.createElement('div');
    document.body.appendChild(root);
    installErrorSurface(root);
  });

  afterEach(() => {
    root.remove();
  });

  it('shows recoverable banner on uncaught error', async () => {
    window.dispatchEvent(new ErrorEvent('error', { message: 'test boom', error: new Error('test boom') }));
    await new Promise((r) => setTimeout(r, 0));
    expect(root.querySelector('.stub-banner')).toBeTruthy();
    expect(root.textContent).toContain('test boom');
    expect(root.querySelector('button')).toBeTruthy();
  });
});
