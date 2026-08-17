/** CPA-M8 consent proof and support feedback DTOs. */

export type ConsentRecordBody = {
  user_id: string;
  purposes: number;
  source: string;
  timestamp?: string;
};

export type ConsentProofDTO = {
  id: number;
  user_id_hash: string;
  purposes: number;
  source: string;
  recorded_at: string;
  ad_storage: boolean;
  analytics_storage: boolean;
};

export type ConsentProofListResponse = {
  items: ConsentProofDTO[];
  next_cursor?: string;
};

export type SupportFeedbackMetaDTO = {
  deployment_id: string;
  binary_version: string;
};

export type SupportFeedbackCreateBody = {
  type: string;
  contact_email: string;
  message: string;
  attach_bundle?: boolean;
};

export type SupportFeedbackCreateResponse = {
  id: string;
};

export type RolesReloadResponse = {
  status: string;
  path: string;
};
