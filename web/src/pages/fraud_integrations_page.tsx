import { FraudIntegrations } from '@/domains/fraud/fraud_integrations';
import { useFraudIntegrationsPageWorkspace } from '@/domains/fraud/use_fraud_integrations_page_workspace';

export function FraudIntegrationsPage() {
  return <FraudIntegrations {...useFraudIntegrationsPageWorkspace()} />;
}
