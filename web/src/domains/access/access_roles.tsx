import { AccessRolesView } from '@/domains/access/access_roles_view';
import { useAccessRolesWorkspace } from '@/domains/access/use_access_roles_workspace';
import { PermissionGate } from '@/shell/permission_gate';

export function AccessRolesPage() {
  const workspace = useAccessRolesWorkspace();
  return (
    <PermissionGate permission="access:read">
      <AccessRolesView {...workspace} />
    </PermissionGate>
  );
}
