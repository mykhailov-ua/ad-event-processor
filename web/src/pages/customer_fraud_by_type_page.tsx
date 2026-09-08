import { CustomerFraudByTypeDirectory } from '@/domains/fraud/customer_fraud_by_type_directory';
import { useCustomerFraudByTypePageWorkspace } from '@/domains/fraud/use_customer_fraud_by_type_page_workspace';

export function CustomerFraudByTypePage() {
  return <CustomerFraudByTypeDirectory {...useCustomerFraudByTypePageWorkspace()} />;
}
