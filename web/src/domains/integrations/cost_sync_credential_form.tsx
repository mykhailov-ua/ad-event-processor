import { ErrorBlock } from '@/shell/error_block';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { CostSyncNetworkSchema } from '@/api/types';

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
    <section className="ui-filter-panel">
      <h2 className="text-base font-semibold">Upsert credentials</h2>
      <p className="text-sm text-muted-foreground">
        Secrets are encrypted at rest. Leave token fields empty to keep existing values on update.
        Click a credentials row below to prefill the network and account fields.
      </p>

      <div className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4">
        <div className="grid gap-2">
          <Label htmlFor="cost-sync-network">Network</Label>
          {networkOptions.length > 0 ? (
            <Select value={draftNetwork} onValueChange={onDraftNetworkChange} disabled={disabled}>
              <SelectTrigger id="cost-sync-network" className="w-full text-sm">
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
        <div className="grid gap-2">
          <Label htmlFor="cost-sync-account-id">Account ID</Label>
          <Input
            id="cost-sync-account-id"
            value={draftAccountId}
            disabled={disabled}
            onChange={(event) => onDraftAccountIdChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="cost-sync-sync-interval">Sync interval (min)</Label>
          <Select
            value={draftSyncIntervalMinutes}
            onValueChange={onDraftSyncIntervalMinutesChange}
            disabled={disabled}
          >
            <SelectTrigger id="cost-sync-sync-interval" className="w-full text-sm">
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
        <div className="grid gap-2 md:col-span-2">
          <Label htmlFor="cost-sync-access-token">Access token</Label>
          <Input
            id="cost-sync-access-token"
            type="password"
            autoComplete="off"
            value={draftAccessToken}
            disabled={disabled}
            onChange={(event) => onDraftAccessTokenChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2 md:col-span-2">
          <Label htmlFor="cost-sync-refresh-token">Refresh token</Label>
          <Input
            id="cost-sync-refresh-token"
            type="password"
            autoComplete="off"
            value={draftRefreshToken}
            disabled={disabled}
            onChange={(event) => onDraftRefreshTokenChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2 md:col-span-2">
          <Label htmlFor="cost-sync-api-key">API key</Label>
          <Input
            id="cost-sync-api-key"
            type="password"
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
      </div>

      {saveError ? <ErrorBlock title="Save failed" message={saveError.message} /> : null}
      {deleteError ? <ErrorBlock title="Delete failed" message={deleteError.message} /> : null}
      {saveSuccess ? (
        <p className="text-sm text-muted-foreground">Credentials saved. List refreshed.</p>
      ) : null}
      {deleteSuccess ? (
        <p className="text-sm text-muted-foreground">Credentials deleted. List refreshed.</p>
      ) : null}
    </section>
  );
}
