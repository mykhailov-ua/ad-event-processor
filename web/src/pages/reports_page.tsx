import { ReportsHub } from '@/domains/reports/reports_hub';
import { useReportsPageWorkspace } from '@/domains/reports/use_reports_page_workspace';

export function ReportsPage() {
  return <ReportsHub {...useReportsPageWorkspace()} />;
}
