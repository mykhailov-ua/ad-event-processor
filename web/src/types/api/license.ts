/** GET /api/v1/license/status */
export type LicenseStatusDTO = {
  deployment_id: string;
  state: string;
  valid_until?: string;
};
