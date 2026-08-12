import { el } from '../lib/dom.js';
import { isAlertDismissed, dismissAlert } from '../helpers/alert_dismiss.js';
import { renderIcon } from './icon.js';

export type AlertBannerOpts = {
  variant?: 'warning' | 'error' | 'info';
  message: string;
  dismissKey?: string;
  onDismiss?: (() => void) | null;
};

/**
 * Render a dismissible alert banner for warnings, errors, or info.
 */
export function renderAlertBanner(opts: AlertBannerOpts): HTMLElement | null {
  if (opts.dismissKey && isAlertDismissed(opts.dismissKey)) return null;

  const variant = opts.variant ?? 'info';
  const iconName = variant === 'error' ? 'alert-circle' : variant === 'warning' ? 'alert-triangle' : 'info';

  return el('div', {
    className: `alert-banner alert-banner--${variant} mb-4`,
    role: opts.variant === 'error' ? 'alert' : 'status',
  },
    renderIcon(iconName, { size: 16, className: 'alert-banner__icon' }),
    el('span', { className: 'alert-banner__text' }, opts.message),
    opts.dismissKey
      ? el('button', {
        type: 'button',
        className: 'alert-banner__close',
        'aria-label': 'Dismiss',
        onClick: () => {
          dismissAlert(opts.dismissKey!);
          if (opts.onDismiss) opts.onDismiss();
        },
      }, renderIcon('x', { size: 14 }))
      : null,
  );
}
