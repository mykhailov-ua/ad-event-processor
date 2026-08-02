import { el } from '../lib/dom.js';

/**
 * Render a license state warning banner when the license is not valid.
 *
 * @param {{ license: { state?: string, valid_until?: string, banner_severity?: string, renew_days?: number } }} opts
 * @returns {HTMLElement}
 */
export function renderLicenseBanner(opts) {
  const license = opts.license;
  if (!license?.state) return el('div');
  const state = license.state.toLowerCase();
  if (state === 'valid' || state === 'active') return el('div');

  const severity = license.banner_severity ?? 'warning';
  const border = severity === 'error' ? 'var(--error)' : 'var(--warning)';

  return el('div', {
    className: 'license-banner',
    style: {
      borderColor: border,
      background: `color-mix(in srgb, ${border} 12%, transparent)`,
    },
  },
    el('span', {}, [
      `License: ${license.state}`,
      license.valid_until ? ` · until ${license.valid_until}` : '',
      license.renew_days != null ? ` · renew ${license.renew_days}d` : '',
    ].filter(Boolean).join('')),
    el('a', { href: '/settings', style: { color: 'var(--accent)', fontSize: 12 } }, 'Settings'),
  );
}
