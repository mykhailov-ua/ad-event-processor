import type { ReactNode } from 'react';
import { Icon } from './icon.js';

export type EmptyStateProps = {
  icon?: string;

  title: string;

  desc?: string;

  action?: ReactNode;

  className?: string;
};

export function EmptyState({ icon = 'tray', title, desc, action, className }: EmptyStateProps) {
  return (
    <div className={`empty-state${className ? ` ${className}` : ''}`}>
      <Icon name={icon} size={32} className="empty-state__icon" />
      <p className="empty-state__title">{title}</p>
      {desc ? <p className="empty-state__desc">{desc}</p> : null}
      {action ? <div className="empty-state__action">{action}</div> : null}
    </div>
  );
}
