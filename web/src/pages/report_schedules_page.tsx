import { ReportSchedulesPanel } from '@/domains/portals/report_schedules_panel';
import { useReportSchedulesPageWorkspace } from '@/domains/reports/use_report_schedules_page_workspace';

export function ReportSchedulesPage() {
  return <ReportSchedulesPanel {...useReportSchedulesPageWorkspace()} />;
}
