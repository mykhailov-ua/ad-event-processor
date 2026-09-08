import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { DirectoryFilterForm, FilterField } from '@/shell/filter_panel';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ErrorBlock } from '@/shell/error_block';
import type { AffiliateStatusPreset } from '@/api/types';
import { IntegrationsNav, integrationsPanelError } from '@/domains/integrations/integrations_nav';

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
  onApplyPreset: (presetName: string) => void;
};

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
  onApplyPreset,
}: IntegrationsAffiliatePresetsProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Affiliate status presets">
        <IntegrationsNav />
        {integrationsPanelError(error, 'Could not load affiliate presets')}
      </PageChrome>
    );
  }

  const canApply = draftCampaignId.trim().length > 0;

  return (
    <PageChrome title="Affiliate status presets">
      <IntegrationsNav />

      <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
        <FilterField className="md:col-span-2" htmlFor="affiliate-preset-campaign-id" label="Campaign ID">
          <Input
            id="affiliate-preset-campaign-id"
            value={draftCampaignId}
            onChange={(event) => onDraftCampaignIdChange(event.target.value)}
            placeholder="Campaign UUID to apply preset"
          />
        </FilterField>
      </DirectoryFilterForm>

      {applyError ? <ErrorBlock title="Apply failed" message={applyError.message} /> : null}
      {applyResult?.mappings_applied_count != null ? (
        <p className="text-sm text-muted-foreground" role="status">
          Last apply: {applyResult.mappings_applied_count} mapping(s) upserted.
        </p>
      ) : null}

      {(presets ?? []).length === 0 ? (
        <EmptyState title="No presets" description="No affiliate status presets are configured." />
      ) : (
        <DirectoryTable>
          <TableHeader>
            <TableRow>
              <DirectoryTableHead>Name</DirectoryTableHead>
              <DirectoryTableHead>Status mappings</DirectoryTableHead>
              <DirectoryTableHead>Apply</DirectoryTableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(presets ?? []).map((row) => (
              <TableRow key={row.name ?? 'preset'}>
                <TableCell>{row.name ?? ''}</TableCell>
                <TableCell>{row.statuses?.length ?? 0}</TableCell>
                <TableCell>
                  <Button
                    disabled={!canApply || applyingPreset != null}
                    onClick={() => onApplyPreset(row.name ?? '')}
                    type="button"
                    variant="outline"
                  >
                    {applyingPreset === row.name ? 'Applying...' : 'Apply to campaign'}
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </DirectoryTable>
      )}

      {error && hasSnapshot ? integrationsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
