import { Navigate, useParams } from 'react-router-dom';
import { isReportLive } from '../models/report.js';
import { ReportRunnerPage } from './report_runner_page.js';
import { ReportStubPage } from '../ui/reports/report_stub_page.js';

export function ReportStubCatchPage() {
  const { reportKey } = useParams<{ reportKey: string }>();
  const key = reportKey ?? '';

  if (key === 'ghost-impression-funnel') {
    return <Navigate to="/reports/silent-reject-impression-funnel" replace />;
  }

  if (isReportLive(key)) {
    return <ReportRunnerPage reportKey={key} />;
  }

  return <ReportStubPage reportKey={key} />;
}
