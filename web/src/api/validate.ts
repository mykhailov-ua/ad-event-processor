// Runtime shape guards for high-traffic list/mutation responses.
// Mismatch throws ApiError(502, INVALID_RESPONSE); callers surface via ErrorBlock.
import { ApiError } from '@/api/api_error';
import type {
  AuditLog,
  Campaign,
  CampaignBulkActionResponse,
  CampaignBulkActionResultRow,
  CampaignFlowValidateResponse,
  CampaignListResponse,
  CampaignPublishBlockedError,
  CampaignValidateResponse,
} from '@/api/types';

export function isRecord(value: unknown): value is Record<string, unknown> {
  return value != null && typeof value === 'object' && !Array.isArray(value);
}

function invalidResponse(message: string): never {
  throw new ApiError(502, 'INVALID_RESPONSE', message);
}

export function parseAuditLogRow(value: unknown): AuditLog {
  if (!isRecord(value)) {
    invalidResponse('Audit log entry must be an object');
  }
  if (typeof value.id !== 'number' || typeof value.action !== 'string') {
    invalidResponse('Audit log entry missing id or action');
  }
  return value as AuditLog;
}

export function parseAuditLogList(value: unknown): AuditLog[] {
  if (!Array.isArray(value)) {
    invalidResponse('Expected JSON array response');
  }
  return value.map(parseAuditLogRow);
}

export function parseCampaign(value: unknown): Campaign {
  if (
    !isRecord(value) ||
    typeof value.id !== 'string' ||
    typeof value.name !== 'string' ||
    typeof value.status !== 'string'
  ) {
    invalidResponse('Campaign missing required fields');
  }
  return value as Campaign;
}

export function parseCampaignListResponse(value: unknown): CampaignListResponse {
  if (!isRecord(value) || !Array.isArray(value.items)) {
    invalidResponse('Campaign list response must include items array');
  }
  if (
    typeof value.total !== 'number' ||
    typeof value.limit !== 'number' ||
    typeof value.offset !== 'number'
  ) {
    invalidResponse('Campaign list response missing pagination fields');
  }
  for (const item of value.items) {
    parseCampaign(item);
  }
  return value as CampaignListResponse;
}

export function parseCampaignBulkActionResultRow(value: unknown): CampaignBulkActionResultRow {
  if (!isRecord(value) || typeof value.id !== 'string' || typeof value.ok !== 'boolean') {
    invalidResponse('Campaign bulk action result row missing id or ok');
  }
  return value as CampaignBulkActionResultRow;
}

export function parseCampaignBulkActionResponse(value: unknown): CampaignBulkActionResponse {
  if (!isRecord(value) || !Array.isArray(value.results)) {
    invalidResponse('Campaign bulk action response must include results array');
  }
  for (const row of value.results) {
    parseCampaignBulkActionResultRow(row);
  }
  return value as CampaignBulkActionResponse;
}

export function parseCampaignPublishBlockedError(value: unknown): CampaignPublishBlockedError {
  if (!isRecord(value) || !isRecord(value.field_errors)) {
    invalidResponse('Publish blocked error missing field_errors');
  }
  return value as CampaignPublishBlockedError;
}

export function parseCampaignFlowValidateResponse(value: unknown): CampaignFlowValidateResponse {
  if (!isRecord(value) || typeof value.valid !== 'boolean') {
    invalidResponse('Flow validate response missing valid boolean');
  }
  return value as CampaignFlowValidateResponse;
}

export function parseCampaignValidateResponse(value: unknown): CampaignValidateResponse {
  if (!isRecord(value) || typeof value.valid !== 'boolean') {
    invalidResponse('Campaign validate response missing valid boolean');
  }
  return value as CampaignValidateResponse;
}
