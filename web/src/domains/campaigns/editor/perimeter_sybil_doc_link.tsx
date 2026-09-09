import { Link } from 'react-router-dom';

export const PERIMETER_SYBIL_DOC_PATH = '';

export function PerimeterSybilDocLink({}: {}) {
  return (
    <Link to={PERIMETER_SYBIL_DOC_PATH}>
      Human Sybil operator limits (perimeter runbook T2)
    </Link>
  );
}
