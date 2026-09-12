import { Input } from '@/components/ui/input';
import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { FilterField, FilterPanel } from '@/shell/filter_panel';
import { PrimaryActionButton } from '@/shell/action_buttons';
import { ErrorBlock } from '@/shell/error_block';
import { cn } from '@/lib/utils';

const FILTER_PANEL_FLAT = 'border-0 bg-transparent p-0 shadow-none';

export type SettingsPersistSectionProps = {
  canWrite: boolean;
  applying: boolean;
  applyError: Error | undefined;
  applyWrittenPath: string | undefined;
  installRoot: string;
  bootstrapComplete: boolean;
  onInstallRootChange: (value: string) => void;
  onApply: () => void;
};

export function SettingsPersistSection({
  canWrite,
  applying,
  applyError,
  applyWrittenPath,
  installRoot,
  bootstrapComplete,
  onInstallRootChange,
  onApply,
}: SettingsPersistSectionProps) {
  return (
    <section className={cn('grid', adminSpacing.gap.lg)}>
      <h2 className={adminTypography.sectionTitle}>Persist to disk</h2>
      <p className={adminTypography.bodyMuted}>
        Writes the current control-plane configuration to install.compose.env on the server. Restart
        tracker and edge services afterward to load the new values.
      </p>
      {applyError ? (
        <ErrorBlock error={applyError} title="Could not write install.compose.env" />
      ) : null}
      {applyWrittenPath ? (
        <p className={adminTypography.body}>
          Last written path: <span className={adminTypography.monoData}>{applyWrittenPath}</span>
        </p>
      ) : null}
      <FilterPanel className={FILTER_PANEL_FLAT}>
        <FilterField htmlFor="settings-install-root" label="Install root (optional)" wide>
          <Input
            disabled={!canWrite || applying}
            id="settings-install-root"
            placeholder="Server default when empty"
            value={installRoot}
            onChange={(event) => onInstallRootChange(event.target.value)}
          />
        </FilterField>
      </FilterPanel>
      <PrimaryActionButton
        disabled={!canWrite || applying || !bootstrapComplete}
        loading={applying}
        type="button"
        variant="destructive"
        onClick={onApply}
      >
        Write install.compose.env
      </PrimaryActionButton>
      {!bootstrapComplete ? (
        <p className={adminTypography.bodyMuted}>
          Platform bootstrap must complete before writing install.compose.env.
        </p>
      ) : null}
    </section>
  );
}
