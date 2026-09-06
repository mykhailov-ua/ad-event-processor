import { CustomersDirectory } from '@/domains/customers/customers_directory';
import { useCustomersPageWorkspace } from '@/domains/customers/use_customers_page_workspace';

export function CustomersPage() {
  return <CustomersDirectory {...useCustomersPageWorkspace()} />;
}
