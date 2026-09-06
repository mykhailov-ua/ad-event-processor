import { FraudPresets } from '@/domains/fraud/fraud_presets';
import { useFraudPresetsPageWorkspace } from '@/domains/fraud/use_fraud_presets_page_workspace';

export function FraudPresetsPage() {
  return <FraudPresets {...useFraudPresetsPageWorkspace()} />;
}
