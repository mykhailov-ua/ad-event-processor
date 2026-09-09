import { Link } from 'react-router-dom';

export const FRAUD_SIGNAL_LIMITS_DOC_PATH = '';

export function FraudLimitsDocLink({}: {}) {
  return (
    <Link to={FRAUD_SIGNAL_LIMITS_DOC_PATH}>
      Fraud signal limits (safe-page, residential proxy, ML batch path)
    </Link>
  );
}
