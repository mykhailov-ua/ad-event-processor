import { AdminErrorPage } from '@/shell/admin_error_page';

export function ForbiddenPage() {
  return <AdminErrorPage kind="forbidden" layout="standalone" />;
}
