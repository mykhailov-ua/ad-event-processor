import type { ReactNode } from 'react';
import { useLocation } from 'react-router-dom';
import * as auth from '../../helpers/auth.js';
import { canAccessRoute } from '../../helpers/route_guard.js';
import { ForbiddenPage } from '../../pages/forbidden_page.js';

export type GuardedRouteProps = {
  children: ReactNode;
};

export function GuardedRoute({ children }: GuardedRouteProps) {
  const location = useLocation();
  const pathname = location.pathname.split('?')[0].replace(/\/$/, '') || '/';
  const user = auth.getUser();
  const permissions = user?.permissions ?? [];

  if (!canAccessRoute(pathname, permissions)) {
    return <ForbiddenPage />;
  }

  return children;
}
