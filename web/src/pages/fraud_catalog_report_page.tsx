import { FraudCatalogReportDirectory } from '@/domains/fraud/fraud_catalog_report_directory';
import type { FraudCatalogReportKey } from '@/domains/fraud/fraud_catalog_report_types';
import { useFraudCatalogReportPageWorkspace } from '@/domains/fraud/use_fraud_catalog_report_page_workspace';

export type FraudCatalogReportPageProps = {
  reportKey: FraudCatalogReportKey;
};

export function FraudCatalogReportPage({ reportKey }: FraudCatalogReportPageProps) {
  return <FraudCatalogReportDirectory {...useFraudCatalogReportPageWorkspace(reportKey)} />;
}
