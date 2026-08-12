/**
 * Go: controlplane.UserDTO — login / boot /me.
 * `/me` may omit email; login usually includes it.
 */
export type AuthUser = {
  id: string;
  email?: string;
  role: string;
  customer_id: string;
  permissions?: string[];
};

/** Go: LoginResponse */
export type LoginResponse = {
  user: AuthUser;
};

/** GET /api/v1/auth/me body */
export type MeResponse = AuthUser;
