import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';
import type { ExportHubCampaignToggleField } from '@/domains/exports/export_hub';
import type { ExportHubKind } from '@/domains/exports/export_hub_catalog';
import type { ExportHubDestination } from '@/domains/exports/export_hub_recent';
import type { ExportHubNotifyChannel } from '@/domains/exports/export_hub_notify_fields';
import type { ExportHubReportFormat } from '@/domains/exports/export_hub_report_formats';

export type ExportHubSavedViewSpec = {
  kind?: ExportHubKind;
  entry?: string;
  from?: string;
  to?: string;
  compare_from?: string;
  compare_to?: string;
  format?: ExportHubReportFormat | 'ndjson';
  destination?: ExportHubDestination;
  row_limit?: number;
  billing_format?: 'csv' | 'ndjson';
  redact_pii?: boolean;
  notify?: {
    channel?: ExportHubNotifyChannel;
    email?: string;
    webhook_url?: string;
  };
  google_sheet?: {
    mode?: 'create' | 'append';
    spreadsheet_id?: string;
    sheet_title?: string;
  };
  import_payload?: Record<string, unknown>;
};

export type ExportHubSavedViewDraft = {
  selectedKind: ExportHubKind;
  entryId?: string;
  reportKey: string;
  from: string;
  to: string;
  compareFrom: string;
  compareTo: string;
  notifyChannel: ExportHubNotifyChannel;
  notifyEmail: string;
  notifyWebhookUrl: string;
  reportFormat: ExportHubReportFormat | '';
  destination: ExportHubDestination;
  rowLimit: string;
  billingFormat: 'csv' | 'ndjson' | '';
  redactPii: boolean;
  spreadsheetId: string;
  sheetTitle: string;
  campaignToggleCampaignId: string;
  campaignToggleField: ExportHubCampaignToggleField | '';
  campaignToggleAt: string;
  campaignToggleWindowHours: string;
  layerDesyncCount: string;
};

function parseSavedViewSpec(raw: unknown): ExportHubSavedViewSpec {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
    return {};
  }
  return raw as ExportHubSavedViewSpec;
}

export function resolveSavedViewReportKey(kind: ExportHubKind, reportKey: string): string {
  if (kind === 'billing') {
    return 'billing-export';
  }
  if (kind === 'audit') {
    return 'audit-export';
  }
  return reportKey.trim() || 'placements';
}

export function buildExportHubSavedViewSpec(draft: ExportHubSavedViewDraft): ExportHubSavedViewSpec {
  const fromIso = fromDatetimeLocalValue(draft.from);
  const toIso = fromDatetimeLocalValue(draft.to);
  const compareFromIso = fromDatetimeLocalValue(draft.compareFrom);
  const compareToIso = fromDatetimeLocalValue(draft.compareTo);
  const rowLimit = Number.parseInt(draft.rowLimit.trim(), 10);

  const spec: ExportHubSavedViewSpec = {
    kind: draft.selectedKind,
  };
  if (draft.entryId) {
    spec.entry = draft.entryId;
  }
  if (fromIso) {
    spec.from = fromIso;
  }
  if (toIso) {
    spec.to = toIso;
  }
  if (compareFromIso) {
    spec.compare_from = compareFromIso;
  }
  if (compareToIso) {
    spec.compare_to = compareToIso;
  }
  if (draft.selectedKind === 'report') {
    if (draft.reportFormat) {
      spec.format = draft.reportFormat;
    }
    spec.destination = draft.destination;
    if (Number.isFinite(rowLimit) && rowLimit > 0) {
      spec.row_limit = rowLimit;
    }
    if (draft.notifyChannel !== 'none') {
      spec.notify = {
        channel: draft.notifyChannel,
        email: draft.notifyEmail.trim() || undefined,
        webhook_url: draft.notifyWebhookUrl.trim() || undefined,
      };
    }
    if (draft.destination === 'google_sheet') {
      const spreadsheetId = draft.spreadsheetId.trim();
      spec.google_sheet = {
        mode: spreadsheetId ? 'append' : 'create',
        spreadsheet_id: spreadsheetId || undefined,
        sheet_title: draft.sheetTitle.trim() || undefined,
      };
    }
    const importPayload = buildImportPayloadFromDraft(draft);
    if (Object.keys(importPayload).length > 0) {
      spec.import_payload = importPayload;
    }
  }
  if (draft.selectedKind === 'billing' && draft.billingFormat) {
    spec.billing_format = draft.billingFormat;
  }
  if (draft.selectedKind === 'audit' && draft.redactPii) {
    spec.redact_pii = true;
  }
  return spec;
}

function buildImportPayloadFromDraft(draft: ExportHubSavedViewDraft): Record<string, unknown> {
  if (draft.reportKey === 'campaign-toggle-cohort') {
    const payload: Record<string, unknown> = {};
    if (draft.campaignToggleCampaignId.trim()) {
      payload.campaign_id = draft.campaignToggleCampaignId.trim();
    }
    if (draft.campaignToggleField) {
      payload.toggle_field = draft.campaignToggleField;
    }
    if (draft.campaignToggleWindowHours.trim()) {
      payload.window_hours = Number.parseInt(draft.campaignToggleWindowHours.trim(), 10);
    }
    const toggleAtIso = fromDatetimeLocalValue(draft.campaignToggleAt);
    if (toggleAtIso) {
      payload.toggle_at = toggleAtIso;
    }
    return payload;
  }
  if (draft.reportKey === 'layer-desync-drilldown' && draft.layerDesyncCount.trim()) {
    return { layer_desync_count: Number.parseInt(draft.layerDesyncCount.trim(), 10) };
  }
  return {};
}

export type ExportHubSavedViewApplyPatch = {
  entryId?: string;
  kind?: ExportHubKind;
  reportKey?: string;
  from?: string;
  to?: string;
  compareFrom?: string;
  compareTo?: string;
  notifyChannel?: ExportHubNotifyChannel;
  notifyEmail?: string;
  notifyWebhookUrl?: string;
  reportFormat?: ExportHubReportFormat | '';
  destination?: ExportHubDestination;
  rowLimit?: string;
  billingFormat?: 'csv' | 'ndjson' | '';
  redactPii?: boolean;
  spreadsheetId?: string;
  sheetTitle?: string;
  campaignToggleCampaignId?: string;
  campaignToggleField?: ExportHubCampaignToggleField | '';
  campaignToggleAt?: string;
  campaignToggleWindowHours?: string;
  layerDesyncCount?: string;
};

export function applyExportHubSavedViewSpec(
  reportKey: string,
  rawSpec: unknown
): ExportHubSavedViewApplyPatch {
  const spec = parseSavedViewSpec(rawSpec);
  const kind = spec.kind ?? (reportKey === 'billing-export' ? 'billing' : reportKey === 'audit-export' ? 'audit' : 'report');
  const patch: ExportHubSavedViewApplyPatch = { kind };

  if (spec.entry) {
    patch.entryId = spec.entry;
  }
  if (kind === 'report') {
    patch.reportKey = reportKey === 'billing-export' || reportKey === 'audit-export' ? 'placements' : reportKey;
  }
  if (spec.from) {
    patch.from = toDatetimeLocalValue(spec.from);
  }
  if (spec.to) {
    patch.to = toDatetimeLocalValue(spec.to);
  }
  if (spec.compare_from) {
    patch.compareFrom = toDatetimeLocalValue(spec.compare_from);
  }
  if (spec.compare_to) {
    patch.compareTo = toDatetimeLocalValue(spec.compare_to);
  }
  if (spec.format === 'csv' || spec.format === 'json' || spec.format === 'xlsx' || spec.format === 'zip') {
    patch.reportFormat = spec.format;
  }
  if (spec.destination === 'google_sheet' || spec.destination === 'download') {
    patch.destination = spec.destination;
  }
  if (spec.row_limit != null && spec.row_limit > 0) {
    patch.rowLimit = String(spec.row_limit);
  }
  if (spec.billing_format === 'csv' || spec.billing_format === 'ndjson') {
    patch.billingFormat = spec.billing_format;
  }
  if (spec.redact_pii) {
    patch.redactPii = true;
  }
  const notifyChannel = spec.notify?.channel;
  if (
    notifyChannel === 'none' ||
    notifyChannel === 'in_app' ||
    notifyChannel === 'email' ||
    notifyChannel === 'slack_webhook'
  ) {
    patch.notifyChannel = notifyChannel;
    patch.notifyEmail = spec.notify?.email ?? '';
    patch.notifyWebhookUrl = spec.notify?.webhook_url ?? '';
  }
  if (spec.google_sheet?.spreadsheet_id) {
    patch.spreadsheetId = spec.google_sheet.spreadsheet_id;
  }
  if (spec.google_sheet?.sheet_title) {
    patch.sheetTitle = spec.google_sheet.sheet_title;
  }
  const payload = spec.import_payload ?? {};
  if (typeof payload.campaign_id === 'string') {
    patch.campaignToggleCampaignId = payload.campaign_id;
  }
  if (
    payload.toggle_field === 'silent_reject_enabled' ||
    payload.toggle_field === 'accept_lang_geo_enabled' ||
    payload.toggle_field === 'json_serialization_enabled'
  ) {
    patch.campaignToggleField = payload.toggle_field;
  }
  if (typeof payload.toggle_at === 'string') {
    patch.campaignToggleAt = toDatetimeLocalValue(payload.toggle_at);
  }
  if (typeof payload.window_hours === 'number') {
    patch.campaignToggleWindowHours = String(payload.window_hours);
  }
  if (typeof payload.layer_desync_count === 'number') {
    patch.layerDesyncCount = String(payload.layer_desync_count);
  }
  return patch;
}
