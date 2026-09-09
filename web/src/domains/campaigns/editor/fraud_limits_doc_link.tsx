import { Link } from 'react-router-dom';

export const FRAUD_SIGNAL_LIMITS_DOC_PATH = '';

const FRAUD_LIMITS_DOC_LABEL =
  'Fraud signal limits (safe-page, residential proxy, ML batch path)';

export function FraudLimitsDocLink({}: {}) {
  if (!FRAUD_SIGNAL_LIMITS_DOC_PATH) {
    return <span className="text-muted-foreground underline-offset-4">{FRAUD_LIMITS_DOC_LABEL}</span>;
  }
  return <Link to={FRAUD_SIGNAL_LIMITS_DOC_PATH}>{FRAUD_LIMITS_DOC_LABEL}</Link>;
}
