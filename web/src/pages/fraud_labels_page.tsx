import { FraudLabels } from '@/domains/fraud/fraud_labels';
import { useFraudLabelsPageWorkspace } from '@/domains/fraud/use_fraud_labels_page_workspace';

export function FraudLabelsPage() {
  return <FraudLabels {...useFraudLabelsPageWorkspace()} />;
}
