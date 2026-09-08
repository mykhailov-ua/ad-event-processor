import { ML_REPORT_CONFIGS, type MlReportKey } from '@/domains/reports/ml_report_meta';
import { MlReportDirectory } from '@/domains/reports/ml_report_directory';
import { useMlReportWorkspace } from '@/domains/reports/use_ml_report_workspace';

function MlReportPageBody<K extends MlReportKey>({ reportKey }: { reportKey: K }) {
  const config = ML_REPORT_CONFIGS[reportKey];
  const workspace = useMlReportWorkspace(config.fetch);
  return <MlReportDirectory config={config} {...workspace} />;
}

export function MlReportPage({ reportKey }: { reportKey: MlReportKey }) {
  return <MlReportPageBody reportKey={reportKey} />;
}
