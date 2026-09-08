import { useParams } from 'react-router-dom';

import { ReportRunner } from '@/domains/reports/report_runner';
import { useReportRunnerPage } from '@/domains/reports/use_report_runner_page';

type ReportRunnerPageProps = {
  reportKey?: string;
};

export function ReportRunnerPage({ reportKey: reportKeyProp }: ReportRunnerPageProps) {
  const { key, segment } = useParams<{ key?: string; segment?: string }>();
  const resolvedKey = reportKeyProp ?? (segment ? `telegram/${segment}` : key ?? '');
  return <ReportRunner {...useReportRunnerPage(resolvedKey)} />;
}
