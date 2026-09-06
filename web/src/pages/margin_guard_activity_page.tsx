import { MarginGuardActivityDirectory } from '@/domains/automation/margin_guard_activity_directory';
import { useMarginGuardActivityPageWorkspace } from '@/domains/automation/use_margin_guard_activity_page_workspace';

export function MarginGuardActivityPage() {
  return <MarginGuardActivityDirectory {...useMarginGuardActivityPageWorkspace()} />;
}
