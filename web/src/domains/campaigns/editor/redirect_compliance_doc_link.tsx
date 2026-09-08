import { Link } from 'react-router-dom';

export const REDIRECT_COMPLIANCE_DOC_PATH = '/docs/tracker';

export function RedirectComplianceDocLink({ className }: { className?: string }) {
  return (
    <Link
      className={className ?? 'text-sm text-muted-foreground hover:underline'}
      to={REDIRECT_COMPLIANCE_DOC_PATH}
    >
      Click redirect profile (strict 302 vs legacy DMR)
    </Link>
  );
}
