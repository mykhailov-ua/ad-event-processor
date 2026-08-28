import type { ButtonHTMLAttributes, ReactNode } from 'react';
import { cn } from '../../lib/cn.js';
import styles from './button.module.css';

export type SystemButtonVariant = 'primary' | 'secondary' | 'danger';

export type SystemButtonProps = {
  variant?: SystemButtonVariant;
  size?: 'sm' | 'md';
  children: ReactNode;
} & Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'>;

export function Button({
  variant = 'secondary',
  size = 'md',
  className,
  type = 'button',
  children,
  ...rest
}: SystemButtonProps) {
  return (
    <button
      type={type}
      className={cn(
        styles.root,
        variant === 'primary' ? styles.primary : '',
        variant === 'danger' ? styles.danger : '',
        size === 'sm' ? styles.sm : '',
        className
      )}
      {...rest}
    >
      {children}
    </button>
  );
}
