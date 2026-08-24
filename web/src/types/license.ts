export type LicenseStatusDTO = {
  deployment_id: string;
  state: string;
  valid_until?: string;
  host_fingerprint?: string;
  hwid_v2?: string;
  hwid_match?: boolean;
  days_to_expiry?: number;
  plan_code?: string;
  max_rps?: number;
  upgrade_plan_code?: string;
  trial_self_serve_url?: string;
  pilot_valid_days?: number;
  support_url?: string;
};

export type ApplyLicenseRequest = {
  token: string;
};
