import { Link } from 'react-router-dom';

export const REDIRECT_COMPLIANCE_DOC_PATH = '';

export function RedirectComplianceDocLink({ }: {}) {
  return (
    <Link
     
      to={REDIRECT_COMPLIANCE_DOC_PATH}
    >
      Click redirect profile (strict 302 vs legacy DMR)
    </Link>
  );
}
