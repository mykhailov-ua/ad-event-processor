import { BillingOverview } from '@/domains/billing/billing_overview';
import { useBillingPageWorkspace } from '@/domains/billing/use_billing_page_workspace';

export function BillingPage() {
  return <BillingOverview {...useBillingPageWorkspace()} />;
}
