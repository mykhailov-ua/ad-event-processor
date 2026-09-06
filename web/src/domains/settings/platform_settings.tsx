import { useMemo } from 'react';

import { PrimaryActionButton } from '@/shell/action_buttons';
import { ErrorBlock } from '@/shell/error_block';
import { FilterField } from '@/shell/filter_panel';
import { PageChrome } from '@/shell/page_chrome';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { COLD_PATH_MAX_BODY_CHARS } from '@/lib/body_limits';
import { settingsFieldLabel } from '@/lib/settings_labels';
import { SettingsBentoGrid } from '@/domains/settings/settings_bento_grid';
import { SettingsCard } from '@/domains/settings/settings_card';
import {
  formatJsonPayloadSize,
  SettingsCollapsibleSection,
} from '@/domains/settings/settings_collapsible_section';
import { SettingsFormActions, SettingsFormStack } from '@/domains/settings/settings_form_stack';
import { settingsHintClass, settingsPageWorkspaceClass } from '@/domains/settings/settings_classes';
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
    [payload]
  );
  const patchDraftReady = draftPatchJson.trim().length > 0;

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={4} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load platform settings" message={error.message} />;
  }

  return (
    <PageChrome
      title="Platform settings"
      workspaceClassName={settingsPageWorkspaceClass}
      badge={
        restartRequired ? (
          <Badge variant="secondary">Restart required</Badge>
        ) : snapshot?.bootstrapComplete ? (
          <Badge variant="outline">Live</Badge>
        ) : undefined
      }
      controlPanel={
        <div className="grid gap-3">
          <SettingsNav />
          <p className={settingsHintClass}>
            Active platform configuration, secrets metadata, and persistence actions.
          </p>
        </div>
      }
    >
      <div className="grid min-h-0 flex-1 auto-rows-max gap-3">
        {showBootstrap ? (
          <SettingsCard title="Initial setup">
            <SettingsFormStack
              onSubmit={(event) => {
                event.preventDefault();
                void onRunBootstrap();
              }}
            >
              <p className={settingsHintClass}>
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
              <FilterField htmlFor="settings-bootstrap-json" label="Setup configuration">
                <Textarea
                  id="settings-bootstrap-json"
                  className="min-h-[10rem] font-mono text-xs"
                  value={draftBootstrapJson}
                  maxLength={COLD_PATH_MAX_BODY_CHARS}
                  onChange={(event) => onDraftBootstrapJsonChange(event.target.value)}
                  placeholder={
                    '{\n  "admin_email": "ops@example.com",\n  "admin_password": "change-me",\n  "config": {\n    "tracking_domain": "track.example.com"\n  }\n}'
                  }
                />
              </FilterField>
              <SettingsFormActions>
                <PrimaryActionButton
                  disabled={
                    bootstrapping || !draftInstallToken.trim() || !draftBootstrapJson.trim()
                  }
                  loading={bootstrapping}
                  type="submit"
                >
                  {bootstrapping ? 'Setting up...' : 'Complete setup'}
                </PrimaryActionButton>
                {bootstrapSuccess ? (
                  <p className={settingsHintClass} role="status">
                    Initial setup completed.
                  </p>
                ) : null}
              </SettingsFormActions>
              {bootstrapError ? (
                <ErrorBlock title="Setup failed" message={bootstrapError.message} />
              ) : null}
            </SettingsFormStack>
          </SettingsCard>
        ) : null}

        {snapshot ? (
          <>
            {snapshot.restartPending.length > 0 ? (
              <SettingsCard
                meta={<Badge variant="secondary">{snapshot.restartPending.length} pending</Badge>}
                title="Pending restart"
              >
                <ul className="m-0 flex list-disc flex-col gap-1 pl-5 text-[13px] leading-[18px] text-muted-foreground">
                  {snapshot.restartPending.map((field) => (
                    <li key={field}>{settingsFieldLabel(field)}</li>
                  ))}
                </ul>
              </SettingsCard>
            ) : null}

            <SettingsCard
              meta={
                snapshot.bootstrapComplete ? (
                  <Badge variant="outline">Ready</Badge>
                ) : (
                  <Badge variant="secondary">Pending</Badge>
                )
              }
              title="Platform configuration"
            >
              <SettingsBentoGrid
                onPatchPlatform={onPatchPlatform}
                patching={patching}
                snapshot={snapshot}
              />
            </SettingsCard>

            <SettingsCollapsibleSection
              badge={`JSON / ${formatJsonPayloadSize(payload)}`}
              title="Raw configuration payload"
            >
              <pre className="ui-code-block max-h-96 overflow-auto text-xs">
                {JSON.stringify(payload, null, 2)}
              </pre>
            </SettingsCollapsibleSection>
          </>
        ) : null}

        <SettingsCollapsibleSection badge="Patch JSON" defaultOpen title="Update configuration">
          <SettingsFormStack
            onSubmit={(event) => {
              event.preventDefault();
              if (patchDraftReady && !patching) {
                void onApplyPatch();
              }
            }}
          >
            <p className={settingsHintClass}>
              Apply partial updates to the active platform configuration. Changes take effect in
              memory; use save to disk when you need a persistent install bundle.
            </p>
            <FilterField htmlFor="settings-patch-json" label="Configuration changes">
              <Textarea
                id="settings-patch-json"
                className="min-h-[8rem] font-mono text-xs"
                value={draftPatchJson}
                maxLength={COLD_PATH_MAX_BODY_CHARS}
                onChange={(event) => onDraftPatchJsonChange(event.target.value)}
                placeholder="{}"
              />
            </FilterField>
            {!patchDraftReady ? (
              <p className={settingsHintClass}>
                Example:{' '}
                <code className="font-mono text-foreground">{`{"tracking_domain":"track.example.com"}`}</code>
              </p>
            ) : null}
            <SettingsFormActions>
              <PrimaryActionButton
                disabled={patching || !patchDraftReady}
                loading={patching}
                type="submit"
              >
                {patching ? 'Applying...' : 'Apply changes'}
              </PrimaryActionButton>
              {patchSuccess ? (
                <p className={settingsHintClass} role="status">
                  Configuration updated.
                </p>
              ) : null}
            </SettingsFormActions>
            {patchError ? (
              <ErrorBlock title="Could not apply changes" message={patchError.message} />
            ) : null}
          </SettingsFormStack>
        </SettingsCollapsibleSection>

        <SettingsCollapsibleSection badge="Disk" title="Save configuration to disk">
          <SettingsFormStack
            onSubmit={(event) => {
              event.preventDefault();
              if (!applying) {
                void onApplyToDisk();
              }
            }}
          >
            <p className={settingsHintClass}>
              Persist the active configuration to disk. Leave the directory empty to use the server
              default install root.
            </p>
            <FilterField htmlFor="settings-install-root" label="Installation directory (optional)">
              <Input
                id="settings-install-root"
                className="font-mono"
                value={draftInstallRoot}
                onChange={(event) => onDraftInstallRootChange(event.target.value)}
                placeholder="/opt/ad-event-processor"
              />
            </FilterField>
            <SettingsFormActions>
              <PrimaryActionButton disabled={applying} loading={applying} type="submit">
                {applying ? 'Saving...' : 'Save to disk'}
              </PrimaryActionButton>
              {applySuccess && applyWrittenPath ? (
                <p className={settingsHintClass} role="status">
                  Saved to <span className="font-mono">{applyWrittenPath}</span>.
                </p>
              ) : null}
            </SettingsFormActions>
            {applyError ? (
              <ErrorBlock title="Could not save to disk" message={applyError.message} />
            ) : null}
          </SettingsFormStack>
        </SettingsCollapsibleSection>

        {error && hasSnapshot ? (
          <ErrorBlock title="Refresh failed" message={error.message} />
        ) : null}
      </div>
    </PageChrome>
  );
}
