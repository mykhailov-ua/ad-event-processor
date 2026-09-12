import { Badge } from '@/components/ui/badge';
import { LicenseApplyForm } from '@/domains/onboarding/license_apply_form';
import type { LicenseApplyFormLoad } from '@/domains/onboarding/use_license_apply_form_load';
import { licenseBadgeVariant } from '@/domains/settings/license_badge';
import { ErrorBlock } from '@/shell/error_block';
import { PageChrome } from '@/shell/page_chrome';
import type { MetaResponse } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { cn } from '@/lib/utils';
export type SettingsLicenseProps = {
  meta: MetaResponse | undefined;
  licenseLoad: LicenseApplyFormLoad;
  stateLabel: string;
  onLicenseApplied: () => void;
  embedded?: boolean;
};

export function SettingsLicense({
  meta,
  licenseLoad,
  stateLabel,
  onLicenseApplied,
  embedded = false,
}: SettingsLicenseProps) {
  const license = meta?.license;
  const validUntil = license?.valid_until?.trim();
  const deploymentId = meta?.deployment_id?.trim();

  const body = (
    <div className={cn('flex max-w-lg flex-col', adminSpacing.gap.xl)}>
      <section className={cn('grid', adminSpacing.gap.lg, adminTypography.body)}>
        <div className={adminSpacing.stack.titleBlock}>
          <span className={adminTypography.labelMuted}>License state</span>
          <span>{stateLabel || 'Unknown'}</span>
        </div>
        {validUntil ? (
          <div className={adminSpacing.stack.titleBlock}>
            <span className={adminTypography.labelMuted}>Valid until</span>
            <span>{displayTimestamp(validUntil)}</span>
          </div>
        ) : null}
        {deploymentId ? (
          <div className={adminSpacing.stack.titleBlock}>
            <span className={adminTypography.labelMuted}>Deployment ID</span>
            <span className={adminTypography.monoData}>{deploymentId}</span>
          </div>
        ) : null}
      </section>

      {licenseLoad.statusError && !licenseLoad.licenseStatus ? (
        <ErrorBlock title="Could not load license status" error={licenseLoad.statusError} />
      ) : null}

      <section className={cn('grid', adminSpacing.gap.lg)}>
        <h2 className={adminTypography.sectionTitle}>Replace license</h2>
        <LicenseApplyForm
          load={licenseLoad}
          showStatus={false}
          description=""
          onApplied={onLicenseApplied}
        />
      </section>
    </div>
  );

  if (embedded) {
    return body;
  }

  return (
    <PageChrome
      title="Settings"
      badge={
        stateLabel ? (
          <Badge variant={licenseBadgeVariant(stateLabel)}>{stateLabel}</Badge>
        ) : undefined
      }
    >
      {body}
    </PageChrome>
  );
}
