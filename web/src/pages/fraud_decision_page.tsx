import { FraudDecisionView } from '@/domains/fraud/fraud_decision';
import { useFraudDecisionPageWorkspace } from '@/domains/fraud/use_fraud_decision_page_workspace';

export function FraudDecisionPage() {
  return <FraudDecisionView {...useFraudDecisionPageWorkspace()} />;
}
