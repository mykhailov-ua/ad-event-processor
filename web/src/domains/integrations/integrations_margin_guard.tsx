import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  IntegrationsPageWithLoad,
  integrationsPanelError,
} from '@/domains/integrations/integrations_nav';
import type { useIntegrationsMarginGuardPageWorkspace } from '@/domains/integrations/use_integrations_margin_guard_page_workspace';
import { CampaignScopeBar } from '@/shell/campaign_scope_bar';
import { DirectoryFilterForm, FilterField, FilterPanel } from '@/shell/filter_panel';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { TableHost } from '@/shell/ui_bands';
import { adminTypography } from '@/lib/admin_kit';
import { displayTimestamp } from '@/lib/display';

export type IntegrationsMarginGuardProps = ReturnType<
  typeof useIntegrationsMarginGuardPageWorkspace
>;

export function IntegrationsMarginGuard(workspace: IntegrationsMarginGuardProps) {
  const {
    canWrite,
    appliedCampaignId,
    draftCampaignId,
    onDraftCampaignIdChange,
    onApplyCampaignScope,
    policies,
    activity,
    fetching,
    error,
    hasSnapshot,
    draftName,
    onDraftNameChange,
    draftMinClicks,
    onDraftMinClicksChange,
    draftRoiFloorPct,
    onDraftRoiFloorPctChange,
    draftZeroConvStreak,
    onDraftZeroConvStreakChange,
    draftCostOverRevenueBps,
    onDraftCostOverRevenueBpsChange,
    creating,
    createError,
    createValidationError,
    onCreatePolicy,
    overridePlacementId,
    onOverridePlacementIdChange,
    removingOverride,
    overrideError,
    onRemoveOverride,
  } = workspace;

  return (
    <IntegrationsPageWithLoad
      alerts={
        <>
          {createError ? integrationsPanelError(createError, 'Create policy failed') : null}
          {overrideError ? integrationsPanelError(overrideError, 'Remove override failed') : null}
        </>
      }
      blockingErrorTitle="Could not load margin guard"
      fetchState={{ error, fetching, hasSnapshot }}
      header={
        <CampaignScopeBar
          appliedCampaignId={appliedCampaignId}
          draftCampaignId={draftCampaignId}
          onApply={onApplyCampaignScope}
          onDraftCampaignIdChange={onDraftCampaignIdChange}
        />
      }
      title="Margin guard"
    >
      <FilterPanel>
        <h2>Create policy</h2>
        <p>
          Requires Pro+ license (`margin_guard` feature). Policies evaluate placement margin
          windows.
        </p>
        {createValidationError ? (
          <ErrorBlock error={createValidationError} title="Check policy fields" />
        ) : null}
        <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
          <FilterField htmlFor="margin-guard-name" label="Name">
            <Input
              id="margin-guard-name"
              value={draftName}
              onChange={(e) => onDraftNameChange(e.target.value)}
            />
          </FilterField>
          <FilterField htmlFor="margin-guard-min-clicks" label="Min clicks">
            <Input
              id="margin-guard-min-clicks"
              value={draftMinClicks}
              onChange={(e) => onDraftMinClicksChange(e.target.value)}
            />
          </FilterField>
          <FilterField htmlFor="margin-guard-roi-floor" label="ROI floor %">
            <Input
              id="margin-guard-roi-floor"
              value={draftRoiFloorPct}
              onChange={(e) => onDraftRoiFloorPctChange(e.target.value)}
            />
          </FilterField>
          <FilterField htmlFor="margin-guard-zero-conv" label="Zero conv streak">
            <Input
              id="margin-guard-zero-conv"
              value={draftZeroConvStreak}
              onChange={(e) => onDraftZeroConvStreakChange(e.target.value)}
            />
          </FilterField>
          <FilterField htmlFor="margin-guard-cost-bps" label="Cost over revenue bps">
            <Input
              id="margin-guard-cost-bps"
              value={draftCostOverRevenueBps}
              onChange={(e) => onDraftCostOverRevenueBpsChange(e.target.value)}
            />
          </FilterField>
          <div className="flex items-end">
            <Button disabled={!canWrite || creating || !appliedCampaignId} onClick={onCreatePolicy}>
              {creating ? 'Creating...' : 'Create policy'}
            </Button>
          </div>
        </DirectoryFilterForm>
        {!canWrite ? (
          <p role="status">Read-only: campaigns:write is required to create policies.</p>
        ) : null}
      </FilterPanel>

      {!appliedCampaignId ? (
        <EmptyState
          description="Apply a campaign ID to list margin guard policies and activity."
          title="Campaign scope required"
        />
      ) : (
        <>
          <TableHost>
            <h2 className={adminTypography.sectionTitle}>Policies</h2>
            {hasSnapshot && policies.length === 0 ? (
              <EmptyState
                description="No margin guard policies for this campaign."
                title="No policies"
              />
            ) : (
              <DirectoryTable>
                <TableHeader>
                  <TableRow>
                    <DirectoryTableHead>Name</DirectoryTableHead>
                    <DirectoryTableHead>Min clicks</DirectoryTableHead>
                    <DirectoryTableHead>ROI floor</DirectoryTableHead>
                    <DirectoryTableHead>Active</DirectoryTableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {policies.map((row) => (
                    <TableRow key={row.id ?? row.name}>
                      <TableCell>{row.name}</TableCell>
                      <TableCell>{row.min_clicks}</TableCell>
                      <TableCell>{row.roi_floor_pct}</TableCell>
                      <TableCell>{row.is_active ? 'Yes' : 'No'}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </DirectoryTable>
            )}
          </TableHost>

          <TableHost>
            <h2 className={adminTypography.sectionTitle}>Recent activity</h2>
            {hasSnapshot && activity.length === 0 ? (
              <EmptyState description="No margin guard actions recorded yet." title="No activity" />
            ) : (
              <DirectoryTable>
                <TableHeader>
                  <TableRow>
                    <DirectoryTableHead>Placement</DirectoryTableHead>
                    <DirectoryTableHead>Action</DirectoryTableHead>
                    <DirectoryTableHead>Reason</DirectoryTableHead>
                    <DirectoryTableHead>Created</DirectoryTableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {activity.map((row) => (
                    <TableRow key={row.id}>
                      <TableCell>{row.placement_id}</TableCell>
                      <TableCell>{row.action}</TableCell>
                      <TableCell>{row.reason}</TableCell>
                      <TableCell>{displayTimestamp(row.created_at)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </DirectoryTable>
            )}
          </TableHost>

          <FilterPanel>
            <h2>Remove placement override</h2>
            <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
              <FilterField htmlFor="margin-guard-placement-id" label="Placement ID">
                <Input
                  id="margin-guard-placement-id"
                  value={overridePlacementId}
                  onChange={(e) => onOverridePlacementIdChange(e.target.value)}
                />
              </FilterField>
              <div className="flex items-end">
                <Button
                  disabled={!canWrite || removingOverride || !appliedCampaignId}
                  variant="secondary"
                  onClick={onRemoveOverride}
                >
                  {removingOverride ? 'Removing...' : 'Remove override'}
                </Button>
              </div>
            </DirectoryFilterForm>
          </FilterPanel>
        </>
      )}
    </IntegrationsPageWithLoad>
  );
}
