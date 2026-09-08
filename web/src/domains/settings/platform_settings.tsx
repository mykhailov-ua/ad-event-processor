import { useMemo } from 'react';

import { PrimaryActionButton } from '@/shell/action_buttons';
import { ErrorBlock } from '@/shell/error_block';
import { FilterField } from '@/shell/filter_panel';
import { PageChrome } from '@/shell/page_chrome';
import { PageSectionStack } from '@/shell/page_layout';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { settingsFieldLabel } from '@/lib/settings_labels';
import { SettingsBentoGrid } from '@/domains/settings/settings_bento_grid';
import {
  SettingsBootstrapForm,
  type SettingsBootstrapDraft,
} from '@/domains/settings/settings_bootstrap_form';
import { SettingsPatchForm } from '@/domains/settings/settings_patch_form';
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
  draftInstallRoot: string;
  draftInstallToken: string;
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
  onDraftInstallRootChange: (value: string) => void;
  onDraftInstallTokenChange: (value: string) => void;
  onPatchPlatform: (patch: Record<string, unknown>) => void;
  onApplyToDisk: () => void;
  onRunBootstrap: (draft: SettingsBootstrapDraft) => void;
};

export function PlatformSettings({
  payload,
  draftInstallRoot,
  draftInstallToken,
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
  onDraftInstallRootChange,
  onDraftInstallTokenChange,
  onPatchPlatform,
  onApplyToDisk,
  onRunBootstrap,
}: PlatformSettingsProps) {
  const snapshot = useMemo(
    () => (payload ? parsePlatformSettingsSnapshot(payload) : undefined),
    [payload]
  );

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
      <PageSectionStack className="min-h-0 flex-1 auto-rows-max">
        {showBootstrap ? (
          <SettingsCard title="Initial setup">
            <SettingsBootstrapForm
              bootstrapping={bootstrapping}
              bootstrapError={bootstrapError}
              bootstrapSuccess={bootstrapSuccess}
              draftInstallToken={draftInstallToken}
              onDraftInstallTokenChange={onDraftInstallTokenChange}
              onRunBootstrap={onRunBootstrap}
            />
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

        <SettingsCollapsibleSection badge="Form" defaultOpen title="Update configuration">
          {snapshot ? (
            <SettingsPatchForm
              onPatchPlatform={onPatchPlatform}
              patchError={patchError}
              patchSuccess={patchSuccess}
              patching={patching}
              snapshot={snapshot}
            />
          ) : null}
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
                  Saved to <span>{applyWrittenPath}</span>.
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
      </PageSectionStack>
    </PageChrome>
  );
}
