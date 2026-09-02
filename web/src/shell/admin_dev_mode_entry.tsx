import { SecondaryActionButton } from '@/shell/action_buttons';
import { enableAdminDevMode, isAdminDevMode } from '@/lib/admin_dev_mode';

export function AdminDevModeEntry() {
  if (isAdminDevMode()) {
    return null;
  }

  return (
    <SecondaryActionButton
      className="w-full"
      type="button"
      onClick={() => {
        enableAdminDevMode();
        window.location.replace('/');
      }}
    >
      Enter dev mode (mock API)
    </SecondaryActionButton>
  );
}
