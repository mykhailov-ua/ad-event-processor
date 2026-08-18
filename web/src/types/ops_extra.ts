export type BlacklistEntryDTO = {
  id?: number | string;
  ip?: string;
  reason?: string;
  source?: string;
  created_at?: string;
  expires_at?: string;
  [key: string]: unknown;
};

export type BlacklistListResponse = {
  items?: BlacklistEntryDTO[];
  total?: number;
  next_cursor?: string;
};

export type OutboxEventDTO = {
  id?: number;
  event_type?: string;
  status?: string;
  created_at?: string;
  [key: string]: unknown;
};

export type OutboxListResponse = {
  items?: OutboxEventDTO[];
  total?: number;
  next_cursor?: string;
};

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

export type DLQEntryDTO = {
  id: string;
  shard_id: number;
  stream_id: string;
  entry_id: string;
  campaign_id?: string;
  event_type?: string;
  error?: string;
  failed_at: string;
  retry_count: number;
  worker_id?: string;

  status?: string;
};

export type FanOutSourceError = {
  source?: string;
  code?: string;
};

export type DLQListResponse = {
  items?: DLQEntryDTO[];
  partial?: boolean;
  errors?: FanOutSourceError[];
  next_cursor?: string;
};
