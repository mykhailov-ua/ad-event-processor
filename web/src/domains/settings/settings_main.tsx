import { Link } from 'react-router-dom';

import { SettingsLicense } from '@/domains/settings/settings_license';
import { SettingsPersistSection } from '@/domains/settings/settings_persist_section';
import { SettingsPlatformForm } from '@/domains/settings/settings_platform_form';
import { SettingsSummary } from '@/domains/settings/settings_summary';
import {
  formatRestartRequiredLabels,
} from '@/domains/settings/settings_field_labels';
import type { SettingsPageWorkspace } from '@/domains/settings/use_settings_page_workspace';
import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { DirectoryPageShell } from '@/shell/directory_page_shell';
import { ErrorBlock } from '@/shell/error_block';
import { SecondaryActionButton } from '@/shell/action_buttons';
import { cn } from '@/lib/utils';

export type SettingsMainProps = SettingsPageWorkspace;

export function SettingsMain(workspace: SettingsMainProps) {
  const {
    meta,
    metaError,
    platformSnapshot,
    draft,
    draftDirty,
    canWrite,
    loading,
    revalidating,
    loadError,
    patchError,
    applyError,
    patching,
    applying,
    applyWrittenPath,
    installRoot,
    onDraftChange,
    onInstallRootChange,
    onSave,
    onDiscard,
    onApply,
    onRefresh,
    licenseLoad,
    stateLabel,
    onLicenseApplied,
  } = workspace;

  const restartRequired = platformSnapshot?.restart_required ?? [];
  const restartLabels = formatRestartRequiredLabels(restartRequired);
  const bootstrapComplete =
    platformSnapshot?.bootstrap_complete ?? meta?.bootstrap_complete ?? false;

  return (
    <DirectoryPageShell
      blockingErrorFooter={
        <SecondaryActionButton type="button" onClick={onRefresh}>
          Retry
        </SecondaryActionButton>
      }
      blockingErrorTitle="Could not load platform settings"
      fetchState={{
        fetching: loading,
        error: loadError,
        hasSnapshot: Boolean(platformSnapshot),
        revalidating,
      }}
      refreshErrorTitle="Settings refresh failed"
      skeletonColumns={2}
      title=""
    >
      <div className={cn('grid', adminSpacing.gap.xl)}>
      {metaError ? (
        <ErrorBlock error={metaError} title="Could not load deployment metadata" />
      ) : null}

      {revalidating ? (
        <p className={adminTypography.bodyMuted} role="status">Refreshing settings…</p>
      ) : null}

      {restartLabels.length > 0 ? (
        <div
          className={cn(
            'rounded-md border border-amber-500/40 bg-amber-500/10',
            adminSpacing.inset.bandLg,
            adminSpacing.gap.md,
            'grid'
          )}
          role="alert"
        >
          <p className={adminTypography.sectionTitle}>Service restart required</p>
          <p className={adminTypography.body}>
            Some changes need a tracker or edge restart after writing to disk:{' '}
            {restartLabels.join(', ')}.
          </p>
        </div>
      ) : null}

      {draftDirty || restartLabels.length > 0 ? (
        <p className={adminTypography.bodyMuted} role="note">
          Changes are saved to the control plane. Write to disk and restart services to apply on the
          tracker.
        </p>
      ) : null}

      <SettingsSummary meta={meta} platformSnapshot={platformSnapshot} />

      <SettingsLicense
        embedded
        licenseLoad={licenseLoad}
        meta={meta}
        stateLabel={stateLabel}
        onLicenseApplied={onLicenseApplied}
      />

      <SettingsPlatformForm
        canWrite={canWrite}
        draft={draft}
        draftDirty={draftDirty}
        maskedSecrets={platformSnapshot?.secrets}
        meta={meta}
        patchError={patchError}
        patching={patching}
        platformSnapshot={platformSnapshot}
        onDiscard={onDiscard}
        onDraftChange={onDraftChange}
        onSave={() => void onSave()}
      />

      <SettingsPersistSection
        applyError={applyError}
        applyWrittenPath={applyWrittenPath}
        applying={applying}
        bootstrapComplete={bootstrapComplete}
        canWrite={canWrite}
        installRoot={installRoot}
        onApply={() => void onApply()}
        onInstallRootChange={onInstallRootChange}
      />

      <p className={adminTypography.bodyMuted}>
        Related:{' '}
        <Link className="text-foreground underline" to="/ops/domains">
          Manage domains
        </Link>
        {' · '}
        <Link className="text-foreground underline" to="/integrations">
          Integrations
        </Link>
        {' · '}
        <Link className="text-foreground underline" to="/docs">
          Documentation
        </Link>
      </p>
      </div>
    </DirectoryPageShell>
  );
}
