export type TeamMemberDTO = {
  user_id: string;
  email: string;
  role: string;
  campaigns_owned: number;
  created_at?: string;
  is_blocked?: boolean;
  spend_cap_micro?: number;
};

export type TeamBudgetApprovalDTO = {
  id: string;
  user_id: string;
  campaign_id: string;
  requested_budget_micro: number;
  previous_budget_micro: number;
  status: string;
  created_at?: string;
};

export type TeamLicenseDTO = {
  state: string;
  valid_until?: string;
  plan_code?: string;
};

export type TeamOverviewDTO = {
  customer_id: string;
  customer_name: string;
  balance_micro?: number;
  currency?: string;
  license?: TeamLicenseDTO | null;
  members: TeamMemberDTO[];
};
