import type { ReactNode } from 'react';
import { NavLink } from 'react-router-dom';

import type { NavGroup } from '@/lib/nav_config';
import { productDisplayName } from '@/lib/product_display_name';

export type ControlPlaneAppFrameProps = {
  navGroups: NavGroup[];
  header?: ReactNode;
  children: ReactNode;
  signingOut?: boolean;
  onSignOut: () => void;
};

function ControlPlaneNav({ groups }: { groups: NavGroup[] }) {
  return (
    <nav aria-label="Main">
      {groups.map((group) => (
        <section key={group.id}>
          <h2>{group.label}</h2>
          <ul>
            {group.items.map((item) => (
              <li key={item.path}>
                <NavLink end={item.path === '/campaigns'} to={item.path}>
                  {item.label}
                </NavLink>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </nav>
  );
}

export function ControlPlaneAppFrame({
  navGroups,
  header,
  children,
  signingOut = false,
  onSignOut,
}: ControlPlaneAppFrameProps) {
  return (
    <div>
      <a href="#main-content">Skip to content</a>
      <div>
        <aside>
          <header>
            <strong>{productDisplayName}</strong>
          </header>
          <ControlPlaneNav groups={navGroups} />
          <button disabled={signingOut} type="button" onClick={onSignOut}>
            {signingOut ? 'Signing out' : 'Sign out'}
          </button>
        </aside>

        <div>
          {header ? <header>{header}</header> : null}
          <main id="main-content" tabIndex={-1}>
            {children}
          </main>
        </div>
      </div>
    </div>
  );
}
