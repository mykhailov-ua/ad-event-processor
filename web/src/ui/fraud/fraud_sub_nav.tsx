import { Link, useLocation } from 'react-router-dom';
import { cn } from '../../lib/cn.js';
import styles from './fraud_shared.module.css';

const FRAUD_LINKS = [
  { to: '/fraud/decisions', label: 'Decisions' },
  { to: '/fraud/labels', label: 'Labels' },
  { to: '/fraud/overrides', label: 'Overrides' },
  { to: '/fraud/presets', label: 'Presets' },
  { to: '/fraud/integrations', label: 'Integrations' },
] as const;

export type FraudSubNavProps = {
  customerId?: string;
};

function buildHref(path: string, customerId?: string): string {
  if (!customerId) return path;
  return `${path}?customer_id=${encodeURIComponent(customerId)}`;
}

export function FraudSubNav({ customerId }: FraudSubNavProps) {
  const location = useLocation();

  return (
    <nav className={styles.subNav} aria-label="Fraud admin">
      {FRAUD_LINKS.map((link) => {
        const active =
          location.pathname === link.to || location.pathname.startsWith(`${link.to}/`);
        return (
          <Link
            key={link.to}
            to={buildHref(link.to, customerId)}
            className={cn(styles.subNavLink, active ? styles.subNavLinkActive : '')}
            aria-current={active ? 'page' : undefined}
          >
            {link.label}
          </Link>
        );
      })}
      <span className={styles.reportLinks}>
        <Link
          to={
            customerId
              ? `/reports/silent-reject-impression-funnel?customer_id=${encodeURIComponent(customerId)}`
              : '/reports/silent-reject-impression-funnel'
          }
          className={styles.reportLink}
        >
          Silent reject funnel report
        </Link>
      </span>
    </nav>
  );
}
