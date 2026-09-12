import { DisputesDirectory } from '@/domains/disputes/disputes_directory';
import { useDisputesPageWorkspace } from '@/domains/disputes/use_disputes_page_workspace';

export function DisputesPage() {
  return <DisputesDirectory {...useDisputesPageWorkspace()} />;
}
