import { Navigate, useLocation, useParams } from 'react-router-dom';

import { ReportRunner } from '@/domains/reports/report_runner';
import { useReportRunnerPage } from '@/domains/reports/use_report_runner_page';
import { typedReportRedirectPath } from '@/lib/report_paths';

type ReportRunnerPageProps = {
  reportKey?: string;
};

export function ReportRunnerPage({ reportKey: reportKeyProp }: ReportRunnerPageProps) {
  const { key, segment } = useParams<{ key?: string; segment?: string }>();
  const location = useLocation();
  const resolvedKey = reportKeyProp ?? (segment ? `telegram/${segment}` : key ?? '');
  const redirectPath = typedReportRedirectPath(resolvedKey);

  if (redirectPath) {
    return <Navigate replace to={`${redirectPath}${location.search}`} />;
  }

  return <ReportRunner {...useReportRunnerPage(resolvedKey)} />;
}
