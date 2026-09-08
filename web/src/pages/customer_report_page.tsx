import {
  CUSTOMER_REPORT_CONFIGS,
  type CustomerReportKey,
} from '@/domains/reports/customer_report_meta';
import { CustomerReportDirectory } from '@/domains/reports/customer_report_directory';
import { useCustomerScopedReportWorkspace } from '@/domains/reports/use_customer_scoped_report_workspace';

export function CustomerReportPage({ reportKey }: { reportKey: CustomerReportKey }) {
  const config = CUSTOMER_REPORT_CONFIGS[reportKey];
  const workspace = useCustomerScopedReportWorkspace(config.fetch);
  return <CustomerReportDirectory config={config} {...workspace} />;
}
