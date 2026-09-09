import { ExportHub } from '@/domains/exports/export_hub';
import { useExportHubPageWorkspace } from '@/domains/exports/use_export_hub_page_workspace';

export function ExportsPage() {
  return <ExportHub {...useExportHubPageWorkspace()} />;
}
