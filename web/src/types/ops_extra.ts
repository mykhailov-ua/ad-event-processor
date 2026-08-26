import type { components } from './generated/openapi.js';

export type BlacklistEntryDTO = components['schemas']['OpsBlacklistEntry'];

export type BlacklistListResponse = components['schemas']['OpsBlacklistListResponse'];

export type OutboxEventDTO = components['schemas']['OutboxEvent'];

export type OutboxListResponse = components['schemas']['OutboxListResponse'];

export type AuditLogRow = {
  id?: string | number;
  action?: string;
  actor?: string;
  admin_id?: string;
  target_type?: string;
  target_id?: string;
  created_at?: string;
  [key: string]: unknown;
};

export type ReconRunDTO = {
  service: string;
  id: number;
  period_start: string;
  period_end: string;
  status: string;
  total_delta?: number;
  campaigns_checked?: number;
  discrepancies_found?: number;
  findings_count?: number;
  intents_checked?: number;
  error_message?: string;
  created_at: string;
  completed_at?: string;
};

export type DLQEntryDTO = components['schemas']['DLQEntry'] & {
  status?: string;
};

export type FanOutSourceError = components['schemas']['FanOutSourceError'];

export type DLQListResponse = components['schemas']['DLQListResponse'];

export type DLQInboxEntryDTO = components['schemas']['DLQInboxEntry'];

export type DLQInboxListResponse = components['schemas']['DLQInboxListResponse'];
