import type { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { cn } from '../../lib/cn.js';
import styles from './selfserve_shared.module.css';

const LINKS = [
  { to: '/selfserve', label: 'Home' },
  { to: '/selfserve/billing', label: 'Billing' },
  { to: '/selfserve/api-keys', label: 'API keys' },
  { to: '/selfserve/campaigns/new', label: 'New campaign' },
];

export type SelfServeShellProps = {
  children: ReactNode;
};

export function SelfServeShell({ children }: SelfServeShellProps) {
  const location = useLocation();
  const path = location.pathname.replace(/\/$/, '') || '/';

  return (
    <div className={styles.root} data-testid="selfserve-shell">
      <nav className={styles.subnav} aria-label="Self-serve">
        {LINKS.map((link) => {
          const active =
            link.to === '/selfserve'
              ? path === '/selfserve'
              : path === link.to || path.startsWith(`${link.to}/`);
          return (
            <Link
              key={link.to}
              to={link.to}
              className={cn(styles.subnavLink, active ? styles.subnavLinkActive : '')}
            >
              {link.label}
            </Link>
          );
        })}
      </nav>
      {children}
    </div>
  );
}
