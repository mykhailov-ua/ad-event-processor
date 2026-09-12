import { useMemo, useState } from 'react';

import { EmptyState } from '@/shell/empty_state';
import { DirectoryFilterForm, FilterField } from '@/shell/filter_panel';
import { Input } from '@/components/ui/input';
import type { AffiliateStatusPreset } from '@/api/types';
import {
  IntegrationsPageWithLoad,
  integrationsPanelError,
} from '@/domains/integrations/integrations_nav';
import { ErrorBlock } from '@/shell/error_block';
import type { DirectoryOverviewField } from '@/shell/directory_overview_dialog';
import {
  DirectorySelectOverviewTable,
  directoryRecordMap,
  directoryOperateRows,
} from '@/shell/directory_select_overview_table';
import { DirectoryRowActionsMenu } from '@/shell/directory_row_actions_menu';
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import { TableHost } from '@/shell/ui_bands';
import type { AdminValidationError } from '@/lib/admin_validation_error';

export type IntegrationsAffiliatePresetsProps = {
  presets?: AffiliateStatusPreset[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftCampaignId: string;
  onDraftCampaignIdChange: (value: string) => void;
  applyingPreset: string | undefined;
  applyError: Error | undefined;
  applyResult: { mappings_applied_count?: number; kind?: string } | undefined;
  formValidationError?: AdminValidationError;
  onApplyPreset: (presetName: string) => void;
};

function affiliatePresetId(row: AffiliateStatusPreset): string {
  return row.name ?? '';
}

function buildAffiliatePresetOverviewFields(row: AffiliateStatusPreset): DirectoryOverviewField[] {
  const statuses = row.statuses ?? [];
  return [
    { label: 'Name', value: row.name ?? '' },
    {
      label: 'Status mappings',
      value:
        statuses.length === 0
          ? '0'
          : statuses
              .map((entry) => `${entry.inbound_status ?? ''} -> ${entry.goal_name ?? ''}`)
              .join(', '),
    },
  ];
}

export function IntegrationsAffiliatePresets({
  presets,
  fetching,
  error,
  hasSnapshot,
  draftCampaignId,
  onDraftCampaignIdChange,
  applyingPreset,
  applyError,
  applyResult,
  formValidationError,
  onApplyPreset,
}: IntegrationsAffiliatePresetsProps) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const canApply = draftCampaignId.trim().length > 0;

  const recordById = useMemo(() => directoryRecordMap(presets, affiliatePresetId), [presets]);
  const rows = useMemo(
    () => directoryOperateRows(presets, affiliatePresetId, (row) => row.name ?? ''),
    [presets]
  );

  return (
    <IntegrationsPageWithLoad
      alerts={applyError ? integrationsPanelError(applyError, 'Apply failed') : null}
      blockingErrorTitle="Could not load affiliate presets"
      fetchState={{ error, fetching, hasSnapshot }}
      title="Affiliate status presets"
    >
      {formValidationError ? (
        <ErrorBlock error={formValidationError} title="Check apply fields" />
      ) : null}
      <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
        <FilterField htmlFor="affiliate-preset-campaign-id" label="Campaign ID">
          <Input
            id="affiliate-preset-campaign-id"
            value={draftCampaignId}
            onChange={(event) => onDraftCampaignIdChange(event.target.value)}
            placeholder="Campaign UUID to apply preset"
          />
        </FilterField>
      </DirectoryFilterForm>

      {applyResult?.mappings_applied_count != null ? (
        <p role="status">Last apply: {applyResult.mappings_applied_count} mapping(s) upserted.</p>
      ) : null}

      {(presets ?? []).length === 0 ? (
        <EmptyState title="No presets" description="No affiliate status presets are configured." />
      ) : (
        <TableHost>
          <DirectorySelectOverviewTable
            buildOverviewFields={buildAffiliatePresetOverviewFields}
            disabled={fetching || applyingPreset != null}
            overviewTitle={(row) => row.name ?? 'Preset'}
            recordById={recordById}
            renderActions={(tableRow, record, openOverview) => (
              <DirectoryRowActionsMenu
                ariaLabel={`Actions for ${String(tableRow.label)}`}
                disabled={fetching || applyingPreset != null}
                onOverview={openOverview}
              >
                <DropdownMenuItem
                  disabled={!canApply || applyingPreset != null || !record.name}
                  onClick={() => onApplyPreset(record.name ?? '')}
                >
                  {applyingPreset === record.name ? 'Applying...' : 'Apply to campaign'}
                </DropdownMenuItem>
              </DirectoryRowActionsMenu>
            )}
            rows={rows}
            selectedId={selectedId}
            onSelectedIdChange={setSelectedId}
          />
        </TableHost>
      )}
    </IntegrationsPageWithLoad>
  );
}
