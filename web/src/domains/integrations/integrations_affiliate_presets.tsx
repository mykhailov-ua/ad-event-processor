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
import type { AffiliateStatusPreset } from '@/api/types';
import { IntegrationsNav, integrationsPanelError } from '@/domains/integrations/integrations_nav';

export type IntegrationsAffiliatePresetsProps = {
  presets: AffiliateStatusPreset[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

export function IntegrationsAffiliatePresets({
  presets,
  fetching,
  error,
  hasSnapshot,
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

  return (
    <PageChrome title="Affiliate status presets">
      <IntegrationsNav />

      {presets.length === 0 ? (
        <EmptyState
          title="No presets"
          description="No affiliate status presets are configured."
        />
      ) : (
        <DirectoryTable>
            <TableHeader>
              <TableRow>
                <DirectoryTableHead>Name</DirectoryTableHead>
                <DirectoryTableHead>Status mappings</DirectoryTableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {presets.map((row) => (
                <TableRow key={row.name ?? 'preset'}>
                  <TableCell>{row.name ?? ''}</TableCell>
                  <TableCell>{row.statuses?.length ?? 0}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </DirectoryTable>
      )}

      {error && hasSnapshot ? integrationsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
