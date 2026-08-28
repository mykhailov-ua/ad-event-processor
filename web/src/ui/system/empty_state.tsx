import type { ReactNode } from 'react';
import { cn } from '../../lib/cn.js';
import styles from './empty_state.module.css';

export type EmptyStateProps = {
  message: string;
  action?: ReactNode;
  className?: string;
};

export function EmptyState({ message, action, className }: EmptyStateProps) {
  return (
    <div className={cn(styles.root, className)}>
      <p className={styles.message}>{message}</p>
      {action ? <div className={styles.action}>{action}</div> : null}
    </div>
  );
}
