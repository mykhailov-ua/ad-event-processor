import type { ReactNode } from 'react';
import { cn } from '../../lib/cn.js';
import styles from './shell.module.css';

export type ShellLayoutProps = {
  sidebar: ReactNode;
  children: ReactNode;
  shellClassName?: string;
  sidebarClassName?: string;
  mainClassName?: string;
};

export function ShellLayout({
  sidebar,
  children,
  shellClassName,
  sidebarClassName,
  mainClassName,
}: ShellLayoutProps) {
  return (
    <div className={cn(styles.shell, shellClassName)}>
      <aside className={cn(styles.sidebar, sidebarClassName)}>{sidebar}</aside>
      <main className={cn(styles.main, mainClassName)}>{children}</main>
    </div>
  );
}
