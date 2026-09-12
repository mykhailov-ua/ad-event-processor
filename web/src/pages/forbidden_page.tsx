import { useSearchParams } from 'react-router-dom';

import { AdminErrorPage } from '@/shell/admin_error_page';

export function ForbiddenPage() {
  const [searchParams] = useSearchParams();
  const requireParam = searchParams.get('require');
  const detail =
    requireParam != null && requireParam.length > 0
      ? `Required permission: ${requireParam}`
      : undefined;

  return <AdminErrorPage detail={detail} kind="forbidden" layout="standalone" />;
}
