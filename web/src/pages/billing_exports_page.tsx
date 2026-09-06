import { BillingExports } from '@/domains/billing/billing_exports';
import { useBillingExportsPageWorkspace } from '@/domains/billing/use_billing_exports_page_workspace';

export function BillingExportsPage() {
  return <BillingExports {...useBillingExportsPageWorkspace()} />;
}
