import { el } from '../lib/dom.js';
import { renderIcon } from './icon.js';

export type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'ghost';
export type ButtonSize = 'sm' | 'md' | 'lg';

type ButtonClassOpts = {
  variant?: ButtonVariant;
  size?: ButtonSize;
  loading?: boolean;
  className?: string;
};

function buttonClasses(opts: ButtonClassOpts): string {
  const variant = opts.variant ?? 'secondary';
  const size = opts.size ?? 'md';
  return [
    'btn',
    `btn--${variant}`,
    size !== 'md' ? `btn--${size}` : '',
    opts.loading ? 'btn--loading' : '',
    opts.className ?? '',
  ].filter(Boolean).join(' ');
}

function buttonChildren(opts: { icon?: string; iconSize?: number; size?: ButtonSize; label: string }) {
  const size = opts.size ?? 'md';
  return [
    opts.icon
      ? renderIcon(opts.icon, { size: opts.iconSize ?? (size === 'sm' ? 14 : 16) })
      : null,
    opts.label,
  ];
}

export type ButtonOpts = {
  label: string;
  variant?: ButtonVariant;
  size?: ButtonSize;
  icon?: string;
  iconSize?: number;
  loading?: boolean;
  disabled?: boolean;
  type?: 'button' | 'submit' | 'reset';
  className?: string;
  testId?: string;
  id?: string;
  title?: string;
  onClick?: (e: Event) => void;
  /** Optional `data-action` for bulk toolbars and analytics hooks. */
  action?: string;
};

/**
 * Render a standardized button (Milestone 4 control sizes + loading state).
 */
export function renderButton(opts: ButtonOpts): HTMLButtonElement {
  const btn = el('button', {
    type: opts.type ?? 'button',
    className: buttonClasses(opts),
    title: opts.title,
    disabled: Boolean(opts.disabled || opts.loading),
    'aria-busy': opts.loading ? 'true' : undefined,
    'data-testid': opts.testId,
    id: opts.id,
    'data-action': opts.action,
    onClick: opts.onClick,
  },
    ...buttonChildren(opts),
  ) as HTMLButtonElement;

  return btn;
}

export type ButtonLinkOpts = {
  label: string;
  href: string;
  variant?: ButtonVariant;
  size?: ButtonSize;
  icon?: string;
  iconSize?: number;
  className?: string;
  testId?: string;
  id?: string;
  title?: string;
  target?: string;
  rel?: string;
  onClick?: (e: Event) => void;
};

/**
 * Render an anchor styled as a button (same tokens as {@link renderButton}).
 */
export function renderButtonLink(opts: ButtonLinkOpts): HTMLAnchorElement {
  const rel = opts.rel ?? (opts.target === '_blank' ? 'noopener noreferrer' : undefined);
  return el('a', {
    href: opts.href,
    className: buttonClasses(opts),
    title: opts.title,
    target: opts.target,
    rel,
    'data-testid': opts.testId,
    id: opts.id,
    onClick: opts.onClick,
  },
    ...buttonChildren(opts),
  ) as HTMLAnchorElement;
}
