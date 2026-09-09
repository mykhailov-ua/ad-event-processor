import { integrationsPanelError } from '@/domains/integrations/integrations_nav';
import { ErrorBlock } from '@/shell/error_block';
import { DirectoryFilterForm, FilterPanel } from '@/shell/filter_panel';
import type { AdminValidationError } from '@/lib/admin_validation_error';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { PlatformCampaignMutation } from '@/api/types';
import { adminTypography } from '@/lib/admin_kit';

export type PlatformCampaignLinkFormProps = {
  disabled: boolean;
  draftCampaignId: string;
  draftNetwork: string;
  draftExternalCampaignId: string;
  draftAccountId: string;
  draftDailyBudgetMicro: string;
  saving: boolean;
  deleting: boolean;
  refreshing: boolean;
  syncing: boolean;
  pausing: boolean;
  resuming: boolean;
  settingBudget: boolean;
  saveError: Error | undefined;
  deleteError: Error | undefined;
  refreshError: Error | undefined;
  syncError: Error | undefined;
  mutationError: Error | undefined;
  saveSuccess: boolean;
  deleteSuccess: boolean;
  refreshSuccess: boolean;
  syncSuccess: boolean;
  mutationResult: PlatformCampaignMutation | undefined;
  formValidationError?: AdminValidationError;
  onDraftCampaignIdChange: (value: string) => void;
  onDraftNetworkChange: (value: string) => void;
  onDraftExternalCampaignIdChange: (value: string) => void;
  onDraftAccountIdChange: (value: string) => void;
  onDraftDailyBudgetMicroChange: (value: string) => void;
  onSave: () => void;
  onDelete: () => void;
  onRefresh: () => void;
  onSyncRun: () => void;
  onPause: () => void;
  onResume: () => void;
  onSetBudget: () => void;
};

export function PlatformCampaignLinkForm({
  disabled,
  draftCampaignId,
  draftNetwork,
  draftExternalCampaignId,
  draftAccountId,
  draftDailyBudgetMicro,
  saving,
  deleting,
  refreshing,
  syncing,
  pausing,
  resuming,
  settingBudget,
  saveError,
  deleteError,
  refreshError,
  syncError,
  mutationError,
  saveSuccess,
  deleteSuccess,
  refreshSuccess,
  syncSuccess,
  mutationResult,
  formValidationError,
  onDraftCampaignIdChange,
  onDraftNetworkChange,
  onDraftExternalCampaignIdChange,
  onDraftAccountIdChange,
  onDraftDailyBudgetMicroChange,
  onSave,
  onDelete,
  onRefresh,
  onSyncRun,
  onPause,
  onResume,
  onSetBudget,
}: PlatformCampaignLinkFormProps) {
  const canSave =
    !disabled &&
    draftCampaignId.trim().length > 0 &&
    draftNetwork.trim().length > 0 &&
    draftExternalCampaignId.trim().length > 0;
  const canLinkAction =
    !disabled && draftCampaignId.trim().length > 0 && draftNetwork.trim().length > 0;
  const canMutate = canLinkAction;
  const canSetBudget = canMutate && draftDailyBudgetMicro.trim().length > 0;

  const canSyncRun = !disabled && draftCampaignId.trim().length > 0;

  return (
    <FilterPanel>
      <h2 className={adminTypography.sectionTitle}>Manage platform campaign links</h2>
      <p className={adminTypography.bodyMuted} >
        Upsert, refresh, or remove external platform links for the applied customer. Pause, resume,
        and budget mutations use a fresh idempotency key per request. Click a link row below to
        prefill campaign and network fields.
      </p>

      {formValidationError ? (
        <ErrorBlock error={formValidationError} title="Check link fields" />
      ) : null}
      <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
        <div>
          <Label htmlFor="platform-campaign-id">Campaign ID</Label>
          <Input
            id="platform-campaign-id"
            value={draftCampaignId}
            disabled={disabled}
            onChange={(event) => onDraftCampaignIdChange(event.target.value)}
          />
        </div>
        <div>
          <Label htmlFor="platform-network">Network</Label>
          <Input
            id="platform-network"
            value={draftNetwork}
            disabled={disabled}
            onChange={(event) => onDraftNetworkChange(event.target.value)}
          />
        </div>
        <div>
          <Label htmlFor="platform-external-campaign-id">External campaign ID</Label>
          <Input
            id="platform-external-campaign-id"
            value={draftExternalCampaignId}
            disabled={disabled}
            onChange={(event) => onDraftExternalCampaignIdChange(event.target.value)}
          />
        </div>
        <div>
          <Label htmlFor="platform-account-id">Account ID (optional)</Label>
          <Input
            id="platform-account-id"
            value={draftAccountId}
            disabled={disabled}
            onChange={(event) => onDraftAccountIdChange(event.target.value)}
          />
        </div>
        <div>
          <Label htmlFor="platform-daily-budget-micro">Daily budget (micro)</Label>
          <Input
            id="platform-daily-budget-micro"
            type="number"
            min={0}
            value={draftDailyBudgetMicro}
            disabled={disabled}
            onChange={(event) => onDraftDailyBudgetMicroChange(event.target.value)}
          />
        </div>
        <Button disabled={saving || !canSave} onClick={onSave} type="button">
          {saving ? 'Saving...' : 'Upsert link'}
        </Button>
        <Button
          disabled={deleting || !canLinkAction}
          onClick={onDelete}
          type="button"
          variant="outline"
        >
          {deleting ? 'Deleting...' : 'Delete link'}
        </Button>
        <Button
          disabled={refreshing || !canLinkAction}
          onClick={onRefresh}
          type="button"
          variant="outline"
        >
          {refreshing ? 'Refreshing...' : 'Refresh link'}
        </Button>
        <Button
          disabled={syncing || !canSyncRun}
          onClick={onSyncRun}
          type="button"
          variant="secondary"
        >
          {syncing ? 'Syncing...' : 'Run platform sync'}
        </Button>
        <Button disabled={pausing || !canMutate} onClick={onPause} type="button" variant="outline">
          {pausing ? 'Pausing...' : 'Pause campaign'}
        </Button>
        <Button
          disabled={resuming || !canMutate}
          onClick={onResume}
          type="button"
          variant="outline"
        >
          {resuming ? 'Resuming...' : 'Resume campaign'}
        </Button>
        <Button
          disabled={settingBudget || !canSetBudget}
          onClick={onSetBudget}
          type="button"
          variant="outline"
        >
          {settingBudget ? 'Setting...' : 'Set budget'}
        </Button>
      </DirectoryFilterForm>

      {saveError ? integrationsPanelError(saveError, 'Upsert failed') : null}
      {deleteError ? integrationsPanelError(deleteError, 'Delete failed') : null}
      {refreshError ? integrationsPanelError(refreshError, 'Refresh failed') : null}
      {syncError ? integrationsPanelError(syncError, 'Platform sync failed') : null}
      {mutationError ? integrationsPanelError(mutationError, 'Campaign mutation failed') : null}
      {saveSuccess ? (
        <p>Link saved. List refreshed.</p>
      ) : null}
      {deleteSuccess ? (
        <p>Link deleted. List refreshed.</p>
      ) : null}
      {refreshSuccess ? (
        <p>Link refreshed. List refreshed.</p>
      ) : null}
      {syncSuccess ? (
        <p>Platform sync completed for campaign.</p>
      ) : null}
      {mutationResult ? (
        <div>
          <p className={adminTypography.bodyMuted} >
            {mutationResult.action}: {mutationResult.status} ({mutationResult.network})
          </p>
          {mutationResult.error_message ? (
            <p className={adminTypography.bodyMuted} >{mutationResult.error_message}</p>
          ) : null}
        </div>
      ) : null}
    </FilterPanel>
  );
}
