import { AdminErrorPage } from '@/shell/admin_error_page';
import { useSession } from '@/hooks/use_session';

export function NotFoundPage() {
  const { authenticated } = useSession();
  return <AdminErrorPage kind="not-found" layout={authenticated ? 'embedded' : 'standalone'} />;
}
