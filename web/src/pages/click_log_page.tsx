import { ClickLogDirectory } from '@/domains/reports/click_log_directory';
import { useClickLogPageWorkspace } from '@/domains/reports/use_click_log_page_workspace';

export function ClickLogPage() {
  return <ClickLogDirectory {...useClickLogPageWorkspace()} />;
}
