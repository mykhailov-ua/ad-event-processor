import { ReportRunner } from '@/domains/reports/report_runner';
import { useReportRunnerPage } from '@/domains/reports/use_report_runner_page';

export function ReportRunnerPage() {
  return <ReportRunner {...useReportRunnerPage()} />;
}
