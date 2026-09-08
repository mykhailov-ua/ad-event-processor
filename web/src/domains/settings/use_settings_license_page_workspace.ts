// license settings page: wraps useLicenseApplyFormLoad; refreshMeta after successful apply.
import { useCallback } from 'react';

import type { LicenseApplyFormLoad } from '@/domains/onboarding/use_license_apply_form_load';
import { useLicenseApplyFormLoad } from '@/domains/onboarding/use_license_apply_form_load';
import { useMeta } from '@/hooks/use_meta';
import { licenseStateLabel } from '@/lib/install_meta';
import type { MetaResponse } from '@/api/types';

export type SettingsLicensePageWorkspace = {
  meta: MetaResponse | undefined;
  licenseLoad: LicenseApplyFormLoad;
  stateLabel: string;
  onLicenseApplied: () => void;
};

export function useSettingsLicensePageWorkspace(): SettingsLicensePageWorkspace {
  const { meta, refreshMeta } = useMeta();
  const licenseLoad = useLicenseApplyFormLoad(false);
  const stateLabel = licenseStateLabel(meta);

  const onLicenseApplied = useCallback(() => {
    refreshMeta();
  }, [refreshMeta]);

  return {
    meta,
    licenseLoad,
    stateLabel,
    onLicenseApplied,
  };
}
