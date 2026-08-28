import type { ReactNode } from 'react';
import { cn } from '../../lib/cn.js';
import styles from './page_chrome.module.css';

export type PageChromeProps = {
  title: ReactNode;
  badge?: ReactNode;
  className?: string;
};

export function PageChrome({ title, badge, className }: PageChromeProps) {
  return (
    <header className={cn(styles.root, className)}>
      <h1 className={styles.title}>{title}</h1>
      {badge ? <div className={styles.badge}>{badge}</div> : null}
    </header>
  );
}
