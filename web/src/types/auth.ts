export type AuthUser = {
  id: string;
  email?: string;
  role: string;
  customer_id: string;
  permissions?: string[];
};
