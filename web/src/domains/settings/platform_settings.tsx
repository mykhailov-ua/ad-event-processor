import { useMemo } from 'react';

import { FilterApplyButton, PrimaryActionButton } from '@/components/system/action_buttons';
import { ErrorBlock } from '@/components/system/error_block';
import { FilterField, FilterPanel } from '@/components/system/filter_panel';
import { PageChrome } from '@/components/system/page_chrome';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { PanelSection } from '@/components/system/stat_panel';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { COLD_PATH_MAX_BODY_CHARS } from '@/lib/body_limits';
import { settingsFieldLabel } from '@/lib/settings_labels';
import { SettingsBentoGrid } from '@/domains/settings/settings_bento_grid';
import {
  formatJsonPayloadSize,
  SettingsCollapsibleSection,
} from '@/domains/settings/settings_collapsible_section';
import { SettingsNav } from '@/domains/settings/settings_nav';
import { parsePlatformSettingsSnapshot } from '@/domains/settings/settings_snapshot';

export type PlatformSettingsProps = {
  payload: Record<string, unknown> | undefined;
  draftPatchJson: string;
  draftInstallRoot: string;
  draftInstallToken: string;
  draftBootstrapJson: string;
  fetching: boolean;
  patching: boolean;
  applying: boolean;
  bootstrapping: boolean;
  error: Error | undefined;
  patchError: Error | undefined;
  applyError: Error | undefined;
  bootstrapError: Error | undefined;
  patchSuccess: boolean;
  applySuccess: boolean;
  bootstrapSuccess: boolean;
  applyWrittenPath: string | undefined;
  hasSnapshot: boolean;
  restartRequired: boolean;
  showBootstrap: boolean;
  onDraftPatchJsonChange: (value: string) => void;
  onDraftInstallRootChange: (value: string) => void;
  onDraftInstallTokenChange: (value: string) => void;
  onDraftBootstrapJsonChange: (value: string) => void;
  onApplyPatch: () => void;
  onPatchPlatform: (patch: Record<string, unknown>) => void;
  onApplyToDisk: () => void;
  onRunBootstrap: () => void;
};

export function PlatformSettings({
  payload,
  draftPatchJson,
  draftInstallRoot,
  draftInstallToken,
  draftBootstrapJson,
  fetching,
  patching,
  applying,
  bootstrapping,
  error,
  patchError,
  applyError,
  bootstrapError,
  patchSuccess,
  applySuccess,
  bootstrapSuccess,
  applyWrittenPath,
  hasSnapshot,
  restartRequired,
  showBootstrap,
  onDraftPatchJsonChange,
  onDraftInstallRootChange,
  onDraftInstallTokenChange,
  onDraftBootstrapJsonChange,
  onApplyPatch,
  onPatchPlatform,
  onApplyToDisk,
  onRunBootstrap,
}: PlatformSettingsProps) {
  const snapshot = useMemo(
    () => (payload ? parsePlatformSettingsSnapshot(payload) : undefined),
    [payload],
  );

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load platform settings" message={error.message} />;
  }

  return (
    <PageChrome
      title="Platform settings"
      description="Active platform configuration, secrets metadata, and persistence actions."
      badge={
        restartRequired ? (
          <Badge variant="secondary">Restart required</Badge>
        ) : snapshot?.bootstrapComplete ? (
          <Badge variant="outline">Live</Badge>
        ) : undefined
      }
    >
      <SettingsNav />

      {showBootstrap ? (
        <PanelSection title="Initial setup">
          <FilterPanel className="rounded-none border-0 bg-transparent">
            <p className="text-sm text-muted-foreground sm:col-span-2">
              Create the platform configuration on first run using the setup token from your
              deployment bundle.
            </p>
            <FilterField htmlFor="settings-install-token" label="Setup token">
              <Input
                id="settings-install-token"
                type="password"
                autoComplete="off"
                value={draftInstallToken}
                onChange={(event) => onDraftInstallTokenChange(event.target.value)}
              />
            </FilterField>
            <FilterField
              className="sm:col-span-2"
              htmlFor="settings-bootstrap-json"
              label="Setup configuration"
            >
              <Textarea
                id="settings-bootstrap-json"
                className="min-h-[10rem] font-mono text-xs"
                value={draftBootstrapJson}
                maxLength={COLD_PATH_MAX_BODY_CHARS}
                onChange={(event) => onDraftBootstrapJsonChange(event.target.value)}
                placeholder={'{\n  "admin_email": "ops@example.com",\n  "admin_password": "change-me",\n  "config": {\n    "tracking_domain": "track.example.com"\n  }\n}'}
              />
            </FilterField>
            <div className="flex flex-wrap items-center gap-3 sm:col-span-2">
              <PrimaryActionButton
                disabled={bootstrapping || !draftInstallToken.trim() || !draftBootstrapJson.trim()}
                loading={bootstrapping}
                onClick={onRunBootstrap}
                type="button"
              >
                {bootstrapping ? 'Setting up...' : 'Complete setup'}
              </PrimaryActionButton>
              {bootstrapSuccess ? (
                <p className="text-sm text-muted-foreground" role="status">
                  Initial setup completed.
                </p>
              ) : null}
            </div>
            {bootstrapError ? (
              <div className="sm:col-span-2">
                <ErrorBlock title="Setup failed" message={bootstrapError.message} />
              </div>
            ) : null}
          </FilterPanel>
        </PanelSection>
      ) : null}

      {snapshot ? (
        <>
          {snapshot.restartPending.length > 0 ? (
            <PanelSection
              meta={<Badge variant="secondary">{snapshot.restartPending.length} pending</Badge>}
              title="Pending restart"
            >
              <ul className="list-disc space-y-1 px-5 pb-5 pl-10 text-sm text-muted-foreground">
                {snapshot.restartPending.map((field) => (
                  <li key={field}>{settingsFieldLabel(field)}</li>
                ))}
              </ul>
            </PanelSection>
          ) : null}

          <PanelSection
            meta={
              snapshot.bootstrapComplete ? (
                <Badge variant="outline">Ready</Badge>
              ) : (
                <Badge variant="secondary">Pending</Badge>
              )
            }
            title="Platform configuration"
          >
            <div className="p-5">
              <SettingsBentoGrid
                onPatchPlatform={onPatchPlatform}
                patching={patching}
                snapshot={snapshot}
              />
            </div>
          </PanelSection>

          <SettingsCollapsibleSection
            badge={`JSON · ${formatJsonPayloadSize(payload)}`}
            title="Raw configuration payload"
          >
            <pre className="ui-code-block max-h-96 overflow-auto text-xs">
              {JSON.stringify(payload, null, 2)}
            </pre>
          </SettingsCollapsibleSection>
        </>
      ) : null}

      <SettingsCollapsibleSection badge="Patch JSON" defaultOpen title="Update configuration">
        <FilterPanel className="rounded-none border-0 bg-transparent p-0">
          <p className="text-sm text-muted-foreground sm:col-span-2">
            Apply partial updates to the active platform configuration. Changes take effect in
            memory; use save to disk when you need a persistent install bundle.
          </p>
          <FilterField
            className="sm:col-span-2"
            htmlFor="settings-patch-json"
            label="Configuration changes"
          >
            <Textarea
              id="settings-patch-json"
              className="min-h-[8rem] font-mono text-xs"
              value={draftPatchJson}
              maxLength={COLD_PATH_MAX_BODY_CHARS}
              onChange={(event) => onDraftPatchJsonChange(event.target.value)}
              placeholder='{"tracking_domain":"track.example.com"}'
            />
          </FilterField>
          <div className="flex flex-wrap items-center gap-3 sm:col-span-2">
            <FilterApplyButton
              disabled={patching || !draftPatchJson.trim()}
              onClick={onApplyPatch}
              type="button"
            >
              {patching ? 'Applying...' : 'Apply changes'}
            </FilterApplyButton>
            {patchSuccess ? (
              <p className="text-sm text-muted-foreground" role="status">
                Configuration updated.
              </p>
            ) : null}
          </div>
          {patchError ? (
            <div className="sm:col-span-2">
              <ErrorBlock title="Could not apply changes" message={patchError.message} />
            </div>
          ) : null}
        </FilterPanel>
      </SettingsCollapsibleSection>

      <SettingsCollapsibleSection badge="Disk" title="Save configuration to disk">
        <FilterPanel className="rounded-none border-0 bg-transparent p-0">
          <p className="text-sm text-muted-foreground sm:col-span-2">
            Persist the active configuration to disk. Leave the directory empty to use the server
            default install root.
          </p>
          <FilterField
            htmlFor="settings-install-root"
            label="Installation directory (optional)"
          >
            <Input
              id="settings-install-root"
              className="font-mono"
              value={draftInstallRoot}
              onChange={(event) => onDraftInstallRootChange(event.target.value)}
              placeholder="/opt/ad-event-processor"
            />
          </FilterField>
          <div className="flex flex-wrap items-center gap-3 sm:col-span-2">
            <PrimaryActionButton disabled={applying} loading={applying} onClick={onApplyToDisk} type="button">
              {applying ? 'Saving...' : 'Save to disk'}
            </PrimaryActionButton>
            {applySuccess && applyWrittenPath ? (
              <p className="text-sm text-muted-foreground" role="status">
                Saved to <span className="font-mono">{applyWrittenPath}</span>.
              </p>
            ) : null}
          </div>
          {applyError ? (
            <div className="sm:col-span-2">
              <ErrorBlock title="Could not save to disk" message={applyError.message} />
            </div>
          ) : null}
        </FilterPanel>
      </SettingsCollapsibleSection>

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageChrome>
  );
}
