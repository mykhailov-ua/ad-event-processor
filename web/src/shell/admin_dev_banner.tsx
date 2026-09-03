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
    <div className="border-b border-amber-200 bg-amber-50 px-3 py-1 text-center text-sm text-amber-900 dark:border-amber-900 dark:bg-amber-950/40 dark:text-amber-200" role="status">
      <span>
        <strong>Dev mode</strong> - mock API responses. UI works without control plane on :8188.
      </span>
      <Button type="button" variant="secondary" onClick={exitDevMode}>
        Exit dev mode
      </Button>
    </div>
  );
}
