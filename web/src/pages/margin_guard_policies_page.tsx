import { MarginGuardPoliciesDirectory } from '@/domains/automation/margin_guard_policies_directory';
import { useMarginGuardPoliciesPageWorkspace } from '@/domains/automation/use_margin_guard_policies_page_workspace';

export function MarginGuardPoliciesPage() {
  return <MarginGuardPoliciesDirectory {...useMarginGuardPoliciesPageWorkspace()} />;
}
