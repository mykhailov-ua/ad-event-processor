import { FraudReasonsDirectory } from '@/domains/fraud/fraud_reasons_directory';
import { useFraudReasonsPageWorkspace } from '@/domains/fraud/use_fraud_reasons_page_workspace';
import type { FraudReasonsReportKey } from '@/api/types';

type FraudReasonsPageProps = {
  reportKey: FraudReasonsReportKey;
};

export function FraudReasonsPage({ reportKey }: FraudReasonsPageProps) {
  return <FraudReasonsDirectory {...useFraudReasonsPageWorkspace(reportKey)} />;
}
