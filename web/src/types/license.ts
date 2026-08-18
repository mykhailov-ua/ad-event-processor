export type LicenseStatusDTO = {
  deployment_id: string;
  state: string;
  valid_until?: string;
  host_fingerprint?: string;
  hwid_v2?: string;
  hwid_match?: boolean;
  days_to_expiry?: number;
};

export type ApplyLicenseRequest = {
  token: string;
};
