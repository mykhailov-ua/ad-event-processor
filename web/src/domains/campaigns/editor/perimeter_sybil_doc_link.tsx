import { Link } from 'react-router-dom';

export const PERIMETER_SYBIL_DOC_PATH = '';

const PERIMETER_SYBIL_DOC_LABEL = 'Human Sybil operator limits (perimeter runbook T2)';

export function PerimeterSybilDocLink({}: {}) {
  if (!PERIMETER_SYBIL_DOC_PATH) {
    return (
      <span className="text-muted-foreground underline-offset-4">{PERIMETER_SYBIL_DOC_LABEL}</span>
    );
  }
  return <Link to={PERIMETER_SYBIL_DOC_PATH}>{PERIMETER_SYBIL_DOC_LABEL}</Link>;
}
