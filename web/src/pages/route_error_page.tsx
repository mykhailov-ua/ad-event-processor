import { isRouteErrorResponse, useRouteError } from 'react-router-dom';

import { AdminErrorPage } from '@/shell/admin_error_page';
import { adminErrorKindFromUnknown } from '@/lib/admin_error';

export function RouteErrorPage({ layout = 'standalone' }: { layout?: 'standalone' | 'embedded' }) {
  const error = useRouteError();
  const kind = isRouteErrorResponse(error) && error.status === 404 ? 'not-found' : adminErrorKindFromUnknown(error);

  return <AdminErrorPage error={error} kind={kind} layout={layout} />;
}
