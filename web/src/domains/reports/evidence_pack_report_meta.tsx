import type { ReactNode } from 'react';

import { getCustomerFraudEvidenceReport, getFraudEvidencePackReport } from '@/api/reports_api';
import type { FraudEvidencePack, ReportRunQuery } from '@/api/types';

export type EvidencePackReportKey = 'customer-fraud-evidence' | 'fraud-evidence-pack';

export type EvidencePackReportConfig = {
  key: EvidencePackReportKey;
  title: string;
  description: string;
  loadReport: (params: ReportRunQuery, signal?: AbortSignal) => Promise<FraudEvidencePack>;
};

export const EVIDENCE_PACK_REPORT_CONFIGS: Record<EvidencePackReportKey, EvidencePackReportConfig> =
  {
    'customer-fraud-evidence': {
      key: 'customer-fraud-evidence',
      title: 'Dispute evidence',
      description: 'Signed redacted evidence bundle for CPA disputes.',
      loadReport: getCustomerFraudEvidenceReport,
    },
    'fraud-evidence-pack': {
      key: 'fraud-evidence-pack',
      title: 'Fraud evidence pack',
      description: 'Signed per-click fraud evidence for operator review.',
      loadReport: getFraudEvidencePackReport,
    },
  };

export type EvidencePackColumn<Row> = {
  id: string;
  label: string;
  cell: (row: Row) => ReactNode;
};

export const EVIDENCE_TIMELINE_COLUMNS: EvidencePackColumn<
  NonNullable<FraudEvidencePack['timeline']>[number]
>[] = [
  { id: 'event_type', label: 'Event', cell: (row) => row.event_type ?? '-' },
  { id: 'created_at', label: 'Created', cell: (row) => row.created_at ?? '-' },
  { id: 'campaign_id', label: 'Campaign', cell: (row) => row.campaign_id ?? '-' },
  { id: 'placement_id', label: 'Placement', cell: (row) => row.placement_id ?? '-' },
  { id: 'country', label: 'Country', cell: (row) => row.country ?? '-' },
  { id: 'sub1', label: 'Sub1', cell: (row) => row.sub1 ?? '-' },
];

export const EVIDENCE_FRAUD_COLUMNS: EvidencePackColumn<
  NonNullable<FraudEvidencePack['fraud_events']>[number]
>[] = [
  { id: 'event_type', label: 'Event', cell: (row) => row.event_type ?? '-' },
  { id: 'fraud_reason', label: 'Reason', cell: (row) => row.fraud_reason ?? '-' },
  { id: 'fraud_score', label: 'Score', cell: (row) => String(row.fraud_score ?? '-') },
  {
    id: 'layer_desync_count',
    label: 'Layer desync',
    cell: (row) => String(row.layer_desync_count ?? '-'),
  },
  {
    id: 'silent_reject_event',
    label: 'Silent reject',
    cell: (row) => (row.silent_reject_event ? 'yes' : 'no'),
  },
  { id: 'created_at', label: 'Created', cell: (row) => row.created_at ?? '-' },
];
