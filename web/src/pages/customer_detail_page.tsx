import { CustomerDetail } from '@/domains/customers/customer_detail';
import { useCustomerDetailPageWorkspace } from '@/domains/customers/use_customer_detail_page_workspace';

export function CustomerDetailPage() {
  return <CustomerDetail {...useCustomerDetailPageWorkspace()} />;
}
