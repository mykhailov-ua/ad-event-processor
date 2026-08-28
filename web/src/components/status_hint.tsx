import type { ReactNode } from 'react';
import { Icon } from './icon.js';

export type StatusHintProps = {
  tone: 'info' | 'error' | 'success';
  message: ReactNode;
  icon?: string;
  className?: string;
};

export function StatusHint({ tone, message, icon, className = '' }: StatusHintProps) {
  const defaultIcon =
    tone === 'error' ? 'alert-circle' : tone === 'success' ? 'check-circle' : 'info';
  const iconName = icon ?? defaultIcon;

  return (
    <div className={`status-hint status-hint--${tone} ${className}`.trim()}>
      <Icon name={iconName} size={16} className="status-hint__icon" />
      <div className="status-hint__message">{message}</div>
    </div>
  );
}
