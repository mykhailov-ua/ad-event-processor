import { PlatformSettings } from '@/domains/settings/platform_settings';
import { useSettingsPageWorkspace } from '@/domains/settings/use_settings_page_workspace';

export function SettingsPage() {
  return <PlatformSettings {...useSettingsPageWorkspace()} />;
}
