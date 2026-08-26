import type { components } from './generated/openapi.js';

export type DataFreshness = components['schemas']['DataFreshness'];

export type ReportCompareDeltas = components['schemas']['ReportCompareDeltas'];

export type ReportRow = components['schemas']['ReportMapRow'];

export type ReportEnvelope<TRow = ReportRow> = {
  rows: TRow[];
  freshness: DataFreshness;
  next_cursor?: string;
};

export type PlacementReportRow = components['schemas']['PlacementReportRow'];

export type KeywordReportRow = components['schemas']['KeywordReportRow'];

export type TrueRoiRow = components['schemas']['TrueRoiReportRow'];

export type ReportJobStatus = components['schemas']['ReportJobStatus'];

export type SavedView = components['schemas']['SavedView'];
