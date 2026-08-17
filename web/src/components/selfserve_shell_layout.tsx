import type { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { Icon } from './icon.js';

const SELF_SERVE_LINKS = [
  { to: '/selfserve', label: 'Portfolio', icon: 'layout-dashboard' },
  { to: '/selfserve/billing', label: 'Billing', icon: 'wallet' },
  { to: '/selfserve/api-keys', label: 'API keys', icon: 'key' },
  { to: '/selfserve/campaigns/new', label: 'Create campaign', icon: 'plus' },
] as const;

export type SelfServeShellLayoutProps = {
  children: ReactNode;
};

/**
 * Reduced buyer shell (G4): portfolio, billing, API keys — no operator nav.
 */
export function SelfServeShellLayout({ children }: SelfServeShellLayoutProps) {
  const location = useLocation();
  const user = auth.getUser();

  return (
    <div className="shell" data-testid="selfserve-shell">
      <nav className="sidebar sidebar--open" style={{ width: 220, minWidth: 220 }}>
        <div className="sidebar__head">
          <Link to="/selfserve" className="sidebar__logo" title="Self-serve portal">
            <Icon name="layers" size={32} className="sidebar__logo-icon" />
            <span className="sidebar__logo-text">Self-serve</span>
          </Link>
        </div>
        <div className="sidebar__body">
          <ul className="sidebar-nav">
            {SELF_SERVE_LINKS.map((link) => {
              const active = location.pathname === link.to
                || (link.to !== '/selfserve' && location.pathname.startsWith(link.to));
              return (
                <li key={link.to}>
                  <Link
                    to={link.to}
                    className={`sidebar-nav__link${active ? ' sidebar-nav__link--active' : ''}`}
                  >
                    <Icon name={link.icon} size={18} />
                    <span>{link.label}</span>
                  </Link>
                </li>
              );
            })}
          </ul>
        </div>
        <div className="sidebar__foot text-muted text-sm">
          {user?.email ?? ''}
        </div>
      </nav>
      <main className="main-content">
        {children}
      </main>
    </div>
  );
}
