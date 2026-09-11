import { SettingsMain } from '@/domains/settings/settings_main';
import { useSettingsPageWorkspace } from '@/domains/settings/use_settings_page_workspace';
import { PermissionGate } from '@/shell/permission_gate';

export function SettingsLicensePage() {
  const workspace = useSettingsPageWorkspace();
  return (
    <PermissionGate permission="settings:read">
      <SettingsMain {...workspace} />
    </PermissionGate>
  );
}
