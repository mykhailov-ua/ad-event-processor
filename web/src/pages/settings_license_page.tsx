import type { ReactNode } from 'react';

import { LicenseApplyForm } from '@/domains/onboarding/license_apply_form';
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
import { Badge } from '@/components/ui/badge';
import { useMeta } from '@/hooks/use_meta';
import { licenseStateLabel } from '@/lib/install_meta';
import { displayTimestamp } from '@/lib/display';

function licenseBadgeVariant(
  state: string,
): 'default' | 'secondary' | 'destructive' | 'outline' {
  const normalized = state.toLowerCase();
  if (normalized === 'active' || normalized === 'valid') {
    return 'default';
  }
  if (normalized === 'trial' || normalized === 'grace') {
    return 'secondary';
  }
  if (normalized === 'expired' || normalized === 'revoked' || normalized === 'missing') {
    return 'destructive';
  }
  return 'outline';
}

function SettingsStatRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className={settingsRowClass}>
      <span className={settingsRowLabelClass}>{label}</span>
      <div className={settingsRowValueClass}>{value}</div>
    </div>
  );
}

export function SettingsLicensePage() {
  const { meta, refreshMeta } = useMeta();
  const stateLabel = licenseStateLabel(meta);
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
        <div className="flex flex-col gap-3">
          <SettingsNav />
          <p className={settingsHintClass}>
            Deployment license state and token replacement for this control plane.
          </p>
        </div>
      }
    >
      <div className="grid gap-3 lg:grid-cols-2">
        <SettingsCard title="License status">
          <div className="rounded-[8px] border border-border bg-card">
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
          <div className="rounded-[8px] border border-border bg-card">
            <SettingsStatRow label="Plan" value={settingsTextValue(license?.plan_code ?? '', 'license_plan')} />
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
            showStatus={false}
            description=""
            onApplied={() => {
              refreshMeta();
            }}
          />
        </SettingsCard>
      </div>
    </PageChrome>
  );
}
