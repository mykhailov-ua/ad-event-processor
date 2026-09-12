import type { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';

import { useSession } from '@/hooks/use_session';
import { AdminErrorPage } from '@/shell/admin_error_page';
import { Button } from '@/components/ui/button';
import { PageSkeleton } from '@/shell/page_skeleton';
import {
  formatRoutePermissionRequirement,
  resolveRoutePermission,
  sessionHasRoutePermission,
  type RoutePermission,
} from '@/lib/route_permissions';

export type PermissionGateProps = RoutePermission & {
  children: ReactNode;
  fallback?: ReactNode;
};

export function ForbiddenPanel({ requiredPermission }: { requiredPermission?: string }) {
  const detail =
    requiredPermission != null && requiredPermission.length > 0
      ? `Required permission: ${requiredPermission}`
      : undefined;

  return (
    <div className="grid gap-4">
      <AdminErrorPage detail={detail} kind="forbidden" layout="embedded" />
      <div>
        <Button asChild type="button" variant="outline">
          <Link to="/">Go home</Link>
        </Button>
      </div>
    </div>
  );
}

export function PermissionGate({
  permission,
  permissionAny,
  children,
  fallback,
}: PermissionGateProps) {
  const { user } = useSession();
  const allowed = sessionHasRoutePermission(user?.permissions, { permission, permissionAny });
  if (!allowed) {
    return fallback ?? <ForbiddenPanel />;
  }
  return children;
}

export function RoutePermissionGuard({ children }: { children: ReactNode }) {
  const { pathname } = useLocation();
  const { user, loading } = useSession();
  const rule = resolveRoutePermission(pathname);
  if (loading && rule) {
    return <PageSkeleton columns={3} variant="directory" />;
  }
  const allowed = sessionHasRoutePermission(user?.permissions, rule);
  if (!allowed) {
    return <ForbiddenPanel requiredPermission={formatRoutePermissionRequirement(rule)} />;
  }
  return children;
}
