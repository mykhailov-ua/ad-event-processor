import { ReportJobs } from '@/domains/reports/report_jobs';
import { useReportJobsPageWorkspace } from '@/domains/reports/use_report_jobs_page_workspace';

export function ReportJobsPage() {
  return <ReportJobs {...useReportJobsPageWorkspace()} />;
}
