import { LicenseApplyForm } from '@/domains/onboarding/license_apply_form';
import { settingsEmptyValue, settingsTextValue } from '@/domains/settings/settings_empty';
import { SettingsNav } from '@/domains/settings/settings_nav';
import { PageChrome } from '@/shell/page_chrome';
import { PanelSection, StatPanel, StatRow } from '@/shell/stat_panel';
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

export function SettingsLicensePage() {
  const { meta, refreshMeta } = useMeta();
  const stateLabel = licenseStateLabel(meta);
  const license = meta?.license;

  return (
    <PageChrome
      title="License"
      description="Deployment license state and token replacement for this control plane."
      badge={
        stateLabel ? (
          <Badge variant={licenseBadgeVariant(stateLabel)}>{stateLabel}</Badge>
        ) : undefined
      }
    >
      <SettingsNav />

      <div className="grid gap-4 sm:grid-cols-2">
        <StatPanel className="h-full" title="License status">
          <StatRow label="State" value={stateLabel} />
          <StatRow
            label="Valid until"
            value={
              license?.valid_until?.trim()
                ? displayTimestamp(license.valid_until)
                : settingsEmptyValue('license_valid_until')
            }
          />
          <StatRow
            label="Deployment"
            value={
              meta?.deployment_id?.trim() ? (
                <span className="break-all font-mono text-xs">{meta.deployment_id}</span>
              ) : (
                settingsEmptyValue('license_deployment')
              )
            }
          />
        </StatPanel>

        <StatPanel className="h-full" title="Entitlements snapshot">
          <StatRow label="Plan" value={settingsTextValue(license?.plan_code ?? '', 'license_plan')} />
          <StatRow
            label="Bootstrap"
            value={meta?.bootstrap_complete ? 'Complete' : 'Pending'}
          />
        </StatPanel>

        <PanelSection className="h-full" title="Apply license">
          <div className="grid gap-4 p-5">
            <p className="text-sm text-muted-foreground">
              Paste the license token from your deployment bundle or vendor portal.
            </p>
            <LicenseApplyForm
              showStatus={false}
              description=""
              onApplied={() => {
                refreshMeta();
              }}
            />
          </div>
        </PanelSection>
      </div>
    </PageChrome>
  );
}
