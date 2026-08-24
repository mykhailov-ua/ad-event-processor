import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import type { CustomerDTO } from '../types/customer.js';
import type { TeamBudgetApprovalDTO, TeamMemberDTO, TeamOverviewDTO } from '../types/team.js';

export async function fetchTeamOverview(customerId?: string): Promise<TeamOverviewDTO> {
  const params = customerId ? `?customer_id=${encodeURIComponent(customerId)}` : '';
  const res = await api<TeamOverviewDTO>(`/api/v1/team/overview${params}`);
  return res.data;
}

export async function updateCustomerCostCenter(
  customerId: string,
  costCenter: string
): Promise<CustomerDTO> {
  const res = await apiConfirmed<CustomerDTO>(
    `/api/v1/customers/${encodeURIComponent(customerId)}/cost-center`,
    {
      method: 'PATCH',
      body: JSON.stringify({ cost_center: costCenter }),
    }
  );
  return res.data;
}

export async function inviteTeamMember(email: string, role: string): Promise<TeamMemberDTO> {
  const res = await apiConfirmed<TeamMemberDTO>('/api/v1/team/members', {
    method: 'POST',
    body: JSON.stringify({ email, role }),
  });
  return res.data;
}

export async function updateTeamMember(
  userId: string,
  body: { role?: string; is_blocked?: boolean; spend_cap_micro?: number }
): Promise<TeamMemberDTO> {
  const res = await apiConfirmed<TeamMemberDTO>(
    `/api/v1/team/members/${encodeURIComponent(userId)}`,
    {
      method: 'PATCH',
      body: JSON.stringify(body),
    }
  );
  return res.data;
}

export async function assignCampaignOwner(campaignId: string, userId: string): Promise<void> {
  await apiConfirmed(`/api/v1/campaigns/${encodeURIComponent(campaignId)}/owner`, {
    method: 'PUT',
    body: JSON.stringify({ user_id: userId }),
  });
}

export async function fetchTeamBudgetApprovals(): Promise<TeamBudgetApprovalDTO[]> {
  const res = await api<TeamBudgetApprovalDTO[]>('/api/v1/team/budget-approvals');
  return res.data ?? [];
}

export async function approveTeamBudget(approvalId: string): Promise<void> {
  await apiConfirmed(`/api/v1/team/budget-approvals/${encodeURIComponent(approvalId)}/approve`, {
    method: 'POST',
    body: '{}',
  });
}

export async function denyTeamBudget(approvalId: string): Promise<void> {
  await apiConfirmed(`/api/v1/team/budget-approvals/${encodeURIComponent(approvalId)}/deny`, {
    method: 'POST',
    body: '{}',
  });
}
