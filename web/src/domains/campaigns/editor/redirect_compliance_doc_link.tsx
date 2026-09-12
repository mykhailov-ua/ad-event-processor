import { Link } from 'react-router-dom';

export const REDIRECT_COMPLIANCE_DOC_PATH = '';

const REDIRECT_COMPLIANCE_DOC_LABEL = 'Click redirect profile (strict 302 vs legacy DMR)';

export function RedirectComplianceDocLink({}: {}) {
  if (!REDIRECT_COMPLIANCE_DOC_PATH) {
    return (
      <span className="text-muted-foreground underline-offset-4">
        {REDIRECT_COMPLIANCE_DOC_LABEL}
      </span>
    );
  }
  return <Link to={REDIRECT_COMPLIANCE_DOC_PATH}>{REDIRECT_COMPLIANCE_DOC_LABEL}</Link>;
}
