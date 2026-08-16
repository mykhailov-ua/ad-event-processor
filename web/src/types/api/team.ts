/** GET /api/v1/team/overview */
export type TeamMemberDTO = {
  user_id: string;
  email: string;
  role: string;
  campaigns_owned: number;
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
