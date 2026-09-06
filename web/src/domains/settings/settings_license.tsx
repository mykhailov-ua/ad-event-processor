import type { ReactNode } from 'react';

import { Badge } from '@/components/ui/badge';
import { LicenseApplyForm } from '@/domains/onboarding/license_apply_form';
import type { LicenseApplyFormLoad } from '@/domains/onboarding/use_license_apply_form_load';
import { licenseBadgeVariant } from '@/domains/settings/license_badge';
import { SettingsCard } from '@/domains/settings/settings_card';
import {
  settingsHintClass,
  settingsPageWorkspaceClass,
  settingsRowClass,
  settingsRowLabelClass,
  settingsRowValueClass,
} from '@/domains/settings/settings_classes';
import { settingsEmptyValue, settingsTextValue } from '@/domains/settings/settings_empty';
import { SettingsNav } from '@/domains/settings/settings_nav';
import { PageChrome } from '@/shell/page_chrome';
import type { MetaResponse } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type SettingsLicenseProps = {
  meta: MetaResponse | undefined;
  licenseLoad: LicenseApplyFormLoad;
  stateLabel: string;
  onLicenseApplied: () => void;
};

function SettingsStatRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className={settingsRowClass}>
      <span className={settingsRowLabelClass}>{label}</span>
      <div className={settingsRowValueClass}>{value}</div>
    </div>
  );
}

export function SettingsLicense({
  meta,
  licenseLoad,
  stateLabel,
  onLicenseApplied,
}: SettingsLicenseProps) {
  const license = meta?.license;

  return (
    <PageChrome
      title="License"
      workspaceClassName={settingsPageWorkspaceClass}
      badge={
        stateLabel ? (
          <Badge variant={licenseBadgeVariant(stateLabel)}>{stateLabel}</Badge>
        ) : undefined
      }
      controlPanel={
        <div className="grid gap-3">
          <SettingsNav />
          <p className={settingsHintClass}>
            Deployment license state and token replacement for this control plane.
          </p>
        </div>
      }
    >
      <div className="grid gap-3 lg:grid-cols-2">
        <SettingsCard title="License status">
          <div className={cn('border border-border bg-card', adminKit.panelRadius)}>
            <SettingsStatRow label="State" value={stateLabel} />
            <SettingsStatRow
              label="Valid until"
              value={
                license?.valid_until?.trim()
                  ? displayTimestamp(license.valid_until)
                  : settingsEmptyValue('license_valid_until')
              }
            />
            <SettingsStatRow
              label="Deployment"
              value={
                meta?.deployment_id?.trim() ? (
                  <span className="break-all font-mono text-xs">{meta.deployment_id}</span>
                ) : (
                  settingsEmptyValue('license_deployment')
                )
              }
            />
          </div>
        </SettingsCard>

        <SettingsCard title="Entitlements snapshot">
          <div className={cn('border border-border bg-card', adminKit.panelRadius)}>
            <SettingsStatRow
              label="Plan"
              value={settingsTextValue(license?.plan_code ?? '', 'license_plan')}
            />
            <SettingsStatRow
              label="Bootstrap"
              value={meta?.bootstrap_complete ? 'Complete' : 'Pending'}
            />
          </div>
        </SettingsCard>

        <SettingsCard className="lg:col-span-2" title="Apply license">
          <p className={settingsHintClass}>
            Paste the license token from your deployment bundle or vendor portal.
          </p>
          <LicenseApplyForm
            load={licenseLoad}
            showStatus={false}
            description=""
            onApplied={onLicenseApplied}
          />
        </SettingsCard>
      </div>
    </PageChrome>
  );
}
