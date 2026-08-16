import type { ButtonHTMLAttributes, ReactNode } from 'react';
import { Icon } from './icon.js';

export type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'ghost';
export type ButtonSize = 'sm' | 'md' | 'lg';

type ButtonClassOpts = {
  variant?: ButtonVariant;
  size?: ButtonSize;
  loading?: boolean;
  className?: string;
};

/**
 * Build shared button class names.
 */
export function buttonClasses(opts: ButtonClassOpts): string {
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

export type ButtonProps = {
  label: string;
  variant?: ButtonVariant;
  size?: ButtonSize;
  icon?: string;
  iconSize?: number;
  loading?: boolean;
  action?: string;
  children?: ReactNode;
} & Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'>;

/**
 * Standardized button (Geist tokens via CSS classes).
 */
export function Button({
  label,
  variant,
  size = 'md',
  icon,
  iconSize,
  loading,
  action,
  className,
  disabled,
  type = 'button',
  ...rest
}: ButtonProps) {
  const iconPx = iconSize ?? (size === 'sm' ? 14 : 16);
  return (
    <button
      type={type}
      className={buttonClasses({ variant, size, loading, className })}
      disabled={Boolean(disabled || loading)}
      aria-busy={loading ? 'true' : undefined}
      data-action={action}
      {...rest}
    >
      {icon ? <Icon name={icon} size={iconPx} /> : null}
      {label}
    </button>
  );
}

export type ButtonLinkProps = {
  label: string;
  href: string;
  variant?: ButtonVariant;
  size?: ButtonSize;
  icon?: string;
  iconSize?: number;
} & Omit<React.AnchorHTMLAttributes<HTMLAnchorElement>, 'children'>;

/**
 * Anchor styled as a button.
 */
export function ButtonLink({
  label,
  href,
  variant,
  size = 'md',
  icon,
  iconSize,
  className,
  target,
  rel,
  ...rest
}: ButtonLinkProps) {
  const iconPx = iconSize ?? (size === 'sm' ? 14 : 16);
  const linkRel = rel ?? (target === '_blank' ? 'noopener noreferrer' : undefined);
  return (
    <a
      href={href}
      className={buttonClasses({ variant, size, className })}
      target={target}
      rel={linkRel}
      {...rest}
    >
      {icon ? <Icon name={icon} size={iconPx} /> : null}
      {label}
    </a>
  );
}
