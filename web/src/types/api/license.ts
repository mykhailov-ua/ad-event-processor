/** GET /api/v1/license/status */
export type LicenseStatusDTO = {
  deployment_id: string;
  state: string;
  valid_until?: string;
  host_fingerprint?: string;
  hwid_v2?: string;
  hwid_match?: boolean;
  days_to_expiry?: number;
};

/** POST /api/v1/license/apply */
export type ApplyLicenseRequest = {
  token: string;
};
