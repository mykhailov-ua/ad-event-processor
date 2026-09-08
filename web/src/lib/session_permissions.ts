/** Client-side permission check aligned with authz.Snapshot.Has (including wildcard). */
export function sessionHasPermission(permissions: string[] | undefined, permission: string): boolean {
  if (permissions === undefined) {
    return true;
  }
  if (permissions.includes('*')) {
    return true;
  }
  return permissions.includes(permission);
}

export function sessionHasAnyPermission(permissions: string[] | undefined, required: string[]): boolean {
  if (permissions === undefined) {
    return true;
  }
  if (permissions.includes('*')) {
    return true;
  }
  return required.some((permission) => permissions.includes(permission));
}
