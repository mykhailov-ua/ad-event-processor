import { Button } from '@/components/ui/button';
import { disableAdminDevMode, isAdminDevMode } from '@/lib/admin_dev_mode';

export function AdminDevBanner() {
  if (!isAdminDevMode()) {
    return null;
  }

  function exitDevMode() {
    disableAdminDevMode();
    window.location.replace('/');
  }

  return (
    <div className="admin-dev-banner" role="status">
      <span>
        <strong>Dev mode</strong> - mock API responses. UI works without control plane on :8188.
      </span>
      <Button type="button" variant="secondary" onClick={exitDevMode}>
        Exit dev mode
      </Button>
    </div>
  );
}
