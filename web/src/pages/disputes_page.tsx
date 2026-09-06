import { DisputesDirectory } from '@/domains/platform/disputes_directory';
import { useDisputesPageWorkspace } from '@/domains/platform/use_disputes_page_workspace';

export function DisputesPage() {
  return <DisputesDirectory {...useDisputesPageWorkspace()} />;
}
