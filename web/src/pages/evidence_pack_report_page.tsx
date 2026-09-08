import {
  EVIDENCE_PACK_REPORT_CONFIGS,
  type EvidencePackReportKey,
} from '@/domains/reports/evidence_pack_report_meta';
import { EvidencePackReportDirectory } from '@/domains/reports/evidence_pack_report_directory';
import { useEvidencePackReportWorkspace } from '@/domains/reports/use_evidence_pack_report_workspace';

export function EvidencePackReportPage({ reportKey }: { reportKey: EvidencePackReportKey }) {
  const config = EVIDENCE_PACK_REPORT_CONFIGS[reportKey];
  const workspace = useEvidencePackReportWorkspace(config);
  return <EvidencePackReportDirectory config={config} {...workspace} />;
}
