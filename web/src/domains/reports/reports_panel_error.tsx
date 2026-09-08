import { panelError } from '@/shell/panel_error';
import { StubBanner } from '@/shell/stub_banner';

export function reportsLicenseStub(featureKey?: string) {
  const label = featureKey?.trim() || 'license';
  return (
    <StubBanner
      title="License required"
      message={`This report requires the ${label} license feature. Contact your operator to enable it.`}
    />
  );
}

export function reportsPanelError(
  error: Error,
  title: string,
  options: { licenseGated?: boolean; featureKey?: string } = {}
) {
  if (options.licenseGated) {
    return reportsLicenseStub(options.featureKey);
  }
  return panelError(error, title);
}
