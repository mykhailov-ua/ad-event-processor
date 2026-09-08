import { Link } from 'react-router-dom';

export const FRAUD_SIGNAL_LIMITS_DOC_PATH = '/docs/fraud-signal-limits';

export function FraudLimitsDocLink({ className }: { className?: string }) {
  return (
    <Link className={className ?? 'text-sm text-muted-foreground hover:underline'} to={FRAUD_SIGNAL_LIMITS_DOC_PATH}>
      Fraud signal limits (safe-page, residential proxy, ML batch path)
    </Link>
  );
}
