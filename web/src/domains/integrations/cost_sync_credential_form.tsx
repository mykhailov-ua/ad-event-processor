import { integrationsPanelError } from '@/domains/integrations/integrations_nav';
import { DirectoryFilterForm, FilterPanel } from '@/shell/filter_panel';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { PasswordInput } from '@/components/ui/password_input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { CostSyncNetworkSchema } from '@/api/types';
import type { AdminValidationError } from '@/lib/admin_validation_error';
import { ValidationErrorBlock } from '@/shell/validation_error_block';
import { adminTypography } from '@/lib/admin_kit';

const SYNC_INTERVAL_OPTIONS = ['15', '30', '60', '1440'] as const;

export type CostSyncCredentialFormProps = {
  networks: CostSyncNetworkSchema[];
  disabled: boolean;
  draftNetwork: string;
  draftAccountId: string;
  draftAccessToken: string;
  draftRefreshToken: string;
  draftApiKey: string;
  draftSyncIntervalMinutes: string;
  saving: boolean;
  deleting: boolean;
  saveError: Error | undefined;
  deleteError: Error | undefined;
  credentialValidationError?: AdminValidationError;
  saveSuccess: boolean;
  deleteSuccess: boolean;
  onDraftNetworkChange: (value: string) => void;
  onDraftAccountIdChange: (value: string) => void;
  onDraftAccessTokenChange: (value: string) => void;
  onDraftRefreshTokenChange: (value: string) => void;
  onDraftApiKeyChange: (value: string) => void;
  onDraftSyncIntervalMinutesChange: (value: string) => void;
  onSave: () => void;
  onDelete: () => void;
};

export function CostSyncCredentialForm({
  networks,
  disabled,
  draftNetwork,
  draftAccountId,
  draftAccessToken,
  draftRefreshToken,
  draftApiKey,
  draftSyncIntervalMinutes,
  saving,
  deleting,
  saveError,
  deleteError,
  credentialValidationError,
  saveSuccess,
  deleteSuccess,
  onDraftNetworkChange,
  onDraftAccountIdChange,
  onDraftAccessTokenChange,
  onDraftRefreshTokenChange,
  onDraftApiKeyChange,
  onDraftSyncIntervalMinutesChange,
  onSave,
  onDelete,
}: CostSyncCredentialFormProps) {
  const networkOptions = networks.map((row) => row.network);
  const canSave = !disabled && draftNetwork.trim().length > 0;
  const canDelete = !disabled && draftNetwork.trim().length > 0;

  return (
    <FilterPanel>
      <h2 className={adminTypography.sectionTitle}>Upsert credentials</h2>
      <p className={adminTypography.bodyMuted} >
        Secrets are encrypted at rest. Leave token fields empty to keep existing values on update.
        Click a credentials row below to prefill the network and account fields.
      </p>

      <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
        <div className="grid gap-2" >
          <Label htmlFor="cost-sync-network">Network</Label>
          {networkOptions.length > 0 ? (
            <Select value={draftNetwork} onValueChange={onDraftNetworkChange} disabled={disabled}>
              <SelectTrigger className="w-full" id="cost-sync-network">
                <SelectValue placeholder="Select network" />
              </SelectTrigger>
              <SelectContent>
                {networkOptions.map((network) => (
                  <SelectItem key={network} value={network}>
                    {network}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : (
            <Input
              id="cost-sync-network"
              value={draftNetwork}
              disabled={disabled}
              onChange={(event) => onDraftNetworkChange(event.target.value)}
            />
          )}
        </div>
        <div className="grid gap-2" >
          <Label htmlFor="cost-sync-account-id">Account ID</Label>
          <Input
            id="cost-sync-account-id"
            value={draftAccountId}
            disabled={disabled}
            onChange={(event) => onDraftAccountIdChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2" >
          <Label htmlFor="cost-sync-sync-interval">Sync interval (min)</Label>
          <Select
            value={draftSyncIntervalMinutes}
            onValueChange={onDraftSyncIntervalMinutesChange}
            disabled={disabled}
          >
            <SelectTrigger className="w-full" id="cost-sync-sync-interval">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {SYNC_INTERVAL_OPTIONS.map((value) => (
                <SelectItem key={value} value={value}>
                  {value}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="grid gap-2 md:col-span-2" >
          <Label htmlFor="cost-sync-access-token">Access token</Label>
          <PasswordInput
            id="cost-sync-access-token"
            autoComplete="off"
            value={draftAccessToken}
            disabled={disabled}
            onChange={(event) => onDraftAccessTokenChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2 md:col-span-2" >
          <Label htmlFor="cost-sync-refresh-token">Refresh token</Label>
          <PasswordInput
            id="cost-sync-refresh-token"
            autoComplete="off"
            value={draftRefreshToken}
            disabled={disabled}
            onChange={(event) => onDraftRefreshTokenChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2 md:col-span-2" >
          <Label htmlFor="cost-sync-api-key">API key</Label>
          <PasswordInput
            id="cost-sync-api-key"
            autoComplete="off"
            value={draftApiKey}
            disabled={disabled}
            onChange={(event) => onDraftApiKeyChange(event.target.value)}
          />
        </div>
        <Button disabled={saving || !canSave} onClick={onSave} type="button">
          {saving ? 'Saving...' : 'Save credentials'}
        </Button>
        <Button
          disabled={deleting || !canDelete}
          onClick={onDelete}
          type="button"
          variant="outline"
        >
          {deleting ? 'Deleting...' : 'Delete credentials'}
        </Button>
      </DirectoryFilterForm>

      {credentialValidationError ? (
        <ValidationErrorBlock error={credentialValidationError} title="Check credential fields" />
      ) : null}
      {saveError ? integrationsPanelError(saveError, 'Save failed') : null}
      {deleteError ? integrationsPanelError(deleteError, 'Delete failed') : null}
      {saveSuccess ? (
        <p>Credentials saved. List refreshed.</p>
      ) : null}
      {deleteSuccess ? (
        <p className={adminTypography.bodyMuted} >Credentials deleted. List refreshed.</p>
      ) : null}
    </FilterPanel>
  );
}
