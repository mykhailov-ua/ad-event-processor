import { Link } from 'react-router-dom';

export const PERIMETER_SYBIL_DOC_PATH = '/docs/perimeter-sybil-controls';

export function PerimeterSybilDocLink({ className }: { className?: string }) {
  return (
    <Link
      className={className ?? 'text-sm text-muted-foreground hover:underline'}
      to={PERIMETER_SYBIL_DOC_PATH}
    >
      Human Sybil operator limits (perimeter runbook T2)
    </Link>
  );
}
