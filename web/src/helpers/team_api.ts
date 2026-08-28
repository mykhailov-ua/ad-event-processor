import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';

export type TeamLicense = {
  state?: string;
  valid_until?: string;
  plan_code?: string;
};

export type TeamMember = {
  user_id?: string;
  email?: string;
  role?: string;
  campaigns_owned?: number;
  created_at?: string;
  is_blocked?: boolean;
  spend_cap_micro?: number;
};

export type TeamOverview = {
  customer_id?: string;
  customer_name?: string;
  cost_center?: string;
  balance_micro?: number;
  currency?: string;
  license?: TeamLicense | null;
  members?: TeamMember[];
};

export type TeamBudgetApproval = {
  id?: string;
  user_id?: string;
  campaign_id?: string;
  requested_budget_micro?: number;
  previous_budget_micro?: number;
  status?: string;
  created_at?: string;
};

export function buildTeamOverviewUrl(customerId: string): string {
  const qs = new URLSearchParams({ customer_id: customerId });
  return `/api/v1/team/overview?${qs.toString()}`;
}

export function buildTeamBudgetApprovalsUrl(customerId: string): string {
  const qs = new URLSearchParams({ customer_id: customerId });
  return `/api/v1/team/budget-approvals?${qs.toString()}`;
}

export async function fetchTeamOverview(
  customerId: string,
  signal?: AbortSignal
): Promise<TeamOverview> {
  const result = await api<TeamOverview>(buildTeamOverviewUrl(customerId), { signal });
  return result.data ?? {};
}

export async function fetchTeamBudgetApprovals(
  customerId: string,
  signal?: AbortSignal
): Promise<TeamBudgetApproval[]> {
  const result = await api<TeamBudgetApproval[]>(buildTeamBudgetApprovalsUrl(customerId), {
    signal,
  });
  return Array.isArray(result.data) ? result.data : [];
}

export async function inviteTeamMember(body: {
  email: string;
  role: string;
  customer_id: string;
}): Promise<TeamMember> {
  const qs = new URLSearchParams({ customer_id: body.customer_id });
  const result = await apiConfirmed<TeamMember>(`/api/v1/team/members?${qs.toString()}`, {
    method: 'POST',
    body: JSON.stringify({ email: body.email, role: body.role }),
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error('invite member failed');
  }
  return result.data ?? {};
}

export async function updateTeamMember(
  userId: string,
  customerId: string,
  body: { role?: string; is_blocked?: boolean; spend_cap_micro?: number }
): Promise<TeamMember> {
  const qs = new URLSearchParams({ customer_id: customerId });
  const result = await apiConfirmed<TeamMember>(
    `/api/v1/team/members/${encodeURIComponent(userId)}?${qs.toString()}`,
    {
      method: 'PATCH',
      body: JSON.stringify(body),
    }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('update member failed');
  }
  return result.data ?? {};
}

export async function approveBudgetApproval(id: string, customerId: string): Promise<void> {
  const qs = new URLSearchParams({ customer_id: customerId });
  const result = await apiConfirmed(
    `/api/v1/team/budget-approvals/${encodeURIComponent(id)}/approve?${qs.toString()}`,
    { method: 'POST', body: '{}' }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('approve budget failed');
  }
}

export async function denyBudgetApproval(id: string, customerId: string): Promise<void> {
  const qs = new URLSearchParams({ customer_id: customerId });
  const result = await apiConfirmed(
    `/api/v1/team/budget-approvals/${encodeURIComponent(id)}/deny?${qs.toString()}`,
    { method: 'POST', body: '{}' }
  );
  if (result.status < 200 || result.status >= 300) {
    throw new Error('deny budget failed');
  }
}
