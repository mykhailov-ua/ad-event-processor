import { SettingsLicense } from '@/domains/settings/settings_license';
import { useSettingsLicensePageWorkspace } from '@/domains/settings/use_settings_license_page_workspace';

export function SettingsPage() {
  return <SettingsLicense {...useSettingsLicensePageWorkspace()} />;
}
