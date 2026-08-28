export type AuthUser = {
  id: string;
  email?: string;
  role: string;
  customer_id: string;
  permissions?: string[];
};

export type LoginResponse = {
  user: AuthUser;
};

export type MeResponse = AuthUser;
