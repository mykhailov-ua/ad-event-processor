import { OpsConsent } from '@/domains/ops/ops_consent';
import { useOpsConsentPageWorkspace } from '@/domains/ops/use_ops_consent_page_workspace';

export function OpsConsentPage() {
  return <OpsConsent {...useOpsConsentPageWorkspace()} />;
}
