import type { ButtonHTMLAttributes, ReactNode } from 'react';
import { cn } from '../../lib/cn.js';
import styles from './button.module.css';

export type SystemButtonVariant = 'primary' | 'secondary' | 'danger';

export type SystemButtonProps = {
  variant?: SystemButtonVariant;
  children: ReactNode;
} & Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'>;

export function Button({
  variant = 'secondary',
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
        className
      )}
      {...rest}
    >
      {children}
    </button>
  );
}
