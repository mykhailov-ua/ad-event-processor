import { FraudOverrides } from '@/domains/fraud/fraud_overrides';
import { useFraudOverridesPageWorkspace } from '@/domains/fraud/use_fraud_overrides_page_workspace';

export function FraudOverridesPage() {
  return <FraudOverrides {...useFraudOverridesPageWorkspace()} />;
}
