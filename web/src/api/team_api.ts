import { apiJson } from './client.js';
import type {
  InviteTeamMemberRequest,
  TeamBudgetApprovalsListResponse,
  TeamBudgetApprovalsQuery,
  TeamMember,
  TeamMembersListResponse,
  TeamMembersQuery,
  TeamOverview,
  TeamOverviewQuery,
  UpdateTeamMemberRequest,
} from './types.js';

export function buildTeamOverviewPath(params: TeamOverviewQuery = {}): string {
  if (!params.customer_id) {
    return '/api/v1/team/overview';
  }
  const search = new URLSearchParams({ customer_id: params.customer_id });
  return `/api/v1/team/overview?${search.toString()}`;
}

export async function getTeamOverview(
  params: TeamOverviewQuery = {},
  signal?: AbortSignal,
): Promise<TeamOverview> {
  return apiJson<TeamOverview>(buildTeamOverviewPath(params), { signal });
}

export async function listTeamMembers(
  params: TeamMembersQuery,
  signal?: AbortSignal,
): Promise<TeamMembersListResponse> {
  const search = new URLSearchParams({ customer_id: params.customer_id });
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }
  return apiJson<TeamMembersListResponse>(`/api/v1/team/members?${search.toString()}`, {
    signal,
  });
}

export async function listTeamBudgetApprovals(
  params: TeamBudgetApprovalsQuery,
  signal?: AbortSignal,
): Promise<TeamBudgetApprovalsListResponse> {
  const search = new URLSearchParams({ customer_id: params.customer_id });
  if (params.limit != null) {
    search.set('limit', String(params.limit));
  }
  if (params.offset != null) {
    search.set('offset', String(params.offset));
  }
  return apiJson<TeamBudgetApprovalsListResponse>(
    `/api/v1/team/budget-approvals?${search.toString()}`,
    { signal },
  );
}

export async function approveTeamBudgetApproval(id: string, signal?: AbortSignal): Promise<void> {
  await apiJson<void>(`/api/v1/team/budget-approvals/${encodeURIComponent(id)}/approve`, {
    method: 'POST',
    signal,
  });
}

export async function denyTeamBudgetApproval(id: string, signal?: AbortSignal): Promise<void> {
  await apiJson<void>(`/api/v1/team/budget-approvals/${encodeURIComponent(id)}/deny`, {
    method: 'POST',
    signal,
  });
}

function withCustomerQuery(path: string, customerId: string): string {
  const search = new URLSearchParams({ customer_id: customerId });
  return `${path}?${search.toString()}`;
}

export async function inviteTeamMember(
  customerId: string,
  body: InviteTeamMemberRequest,
  signal?: AbortSignal,
): Promise<TeamMember> {
  return apiJson<TeamMember>(withCustomerQuery('/api/v1/team/members', customerId), {
    method: 'POST',
    body: JSON.stringify(body),
    signal,
  });
}

export async function updateTeamMember(
  customerId: string,
  memberId: string,
  body: UpdateTeamMemberRequest,
  signal?: AbortSignal,
): Promise<TeamMember> {
  return apiJson<TeamMember>(
    withCustomerQuery(`/api/v1/team/members/${encodeURIComponent(memberId)}`, customerId),
    {
      method: 'PATCH',
      body: JSON.stringify(body),
      signal,
    },
  );
}
