import { Copy } from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { APIKeyCreatedResponse, APIKeySummary } from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { EmptyState } from '@/shell/empty_state';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { RowActionsMenu } from '@/shell/row_actions_menu';
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import {
  IntegrationsPageWithLoad,
  integrationsPanelError,
} from '@/domains/integrations/integrations_nav';
import {
  SELF_SERVE_API_KEY_SCOPES,
  type useIntegrationsApiKeysPageWorkspace,
} from '@/domains/integrations/use_integrations_api_keys_page_workspace';

export type IntegrationsApiKeysProps = ReturnType<typeof useIntegrationsApiKeysPageWorkspace>;

async function copyText(value: string) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(value);
    return;
  }
  const textarea = document.createElement('textarea');
  textarea.value = value;
  textarea.setAttribute('readonly', 'true');
  document.body.appendChild(textarea);
  textarea.select();
  document.execCommand('copy');
  document.body.removeChild(textarea);
}

export function IntegrationsApiKeys({
  keys,
  fetching,
  listRevalidating,
  error,
  hasSnapshot,
  draftName,
  draftScopes,
  creating,
  createError,
  createdKey,
  revokingId,
  revokeError,
  onDraftNameChange,
  onToggleScope,
  onCreate,
  onRevoke,
  onDismissCreatedKey,
}: IntegrationsApiKeysProps) {
  return (
    <IntegrationsPageWithLoad
      blockingErrorTitle="Could not load service accounts"
      fetchState={{ fetching, error, hasSnapshot, revalidating: listRevalidating }}
      title="Service accounts"
      alerts={
        <>
          {createError ? integrationsPanelError(createError, 'Create failed') : null}
          {revokeError ? integrationsPanelError(revokeError, 'Revoke failed') : null}
        </>
      }
    >
      <section>
        <h2>Create service account key</h2>
        <div>
          <Label htmlFor="api-key-name">Name</Label>
          <Input
            id="api-key-name"
            disabled={creating}
            onChange={(event) => onDraftNameChange(event.target.value)}
            placeholder="Integration automation"
            value={draftName}
          />
        </div>
        <fieldset disabled={creating}>
          <legend>Scopes</legend>
          {SELF_SERVE_API_KEY_SCOPES.map((scope) => (
            <label key={scope}>
              <Checkbox
                checked={draftScopes.includes(scope)}
                onCheckedChange={(checked) => onToggleScope(scope, checked === true)}
              />
              {scope}
            </label>
          ))}
        </fieldset>
        <Button disabled={creating || draftName.trim() === ''} onClick={onCreate} type="button">
          {creating ? 'Creating...' : 'Create Bearer key'}
        </Button>
      </section>

      <section>
        <h2>Active keys</h2>
        {keys.length === 0 ? (
          <EmptyState
            description="Create a Bearer key for automation tools. Keys stay active when a team member is offboarded."
            title="No service account keys"
          />
        ) : (
          <DirectoryTable horizontalScroll>
            <TableHeader>
              <TableRow>
                <DirectoryTableHead>Name</DirectoryTableHead>
                <DirectoryTableHead>Scopes</DirectoryTableHead>
                <DirectoryTableHead>Created</DirectoryTableHead>
                <DirectoryTableHead>Expires</DirectoryTableHead>
                <DirectoryTableHead>Actions</DirectoryTableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {keys.map((row) => (
                <ApiKeyRow
                  key={row.id}
                  disabled={fetching || revokingId != null}
                  onRevoke={onRevoke}
                  revoking={revokingId === row.id}
                  row={row}
                />
              ))}
            </TableBody>
          </DirectoryTable>
        )}
      </section>

      <CreatedKeyDialog createdKey={createdKey} onDismiss={onDismissCreatedKey} />
    </IntegrationsPageWithLoad>
  );
}

function ApiKeyRow({
  row,
  disabled,
  revoking,
  onRevoke,
}: {
  row: APIKeySummary;
  disabled: boolean;
  revoking: boolean;
  onRevoke: (row: APIKeySummary) => void;
}) {
  return (
    <TableRow>
      <TableCell>{row.name}</TableCell>
      <TableCell>
        {(row.scopes ?? []).map((scope) => (
          <Badge key={scope} variant="secondary">
            {scope}
          </Badge>
        ))}
      </TableCell>
      <TableCell>{displayTimestamp(row.created_at)}</TableCell>
      <TableCell>{row.expires_at ? displayTimestamp(row.expires_at) : ''}</TableCell>
      <TableCell>
        <RowActionsMenu ariaLabel="API key actions" disabled={disabled || !row.id}>
          <DropdownMenuItem disabled={disabled || revoking || !row.id} onClick={() => onRevoke(row)}>
            {revoking ? 'Revoking...' : 'Revoke'}
          </DropdownMenuItem>
        </RowActionsMenu>
      </TableCell>
    </TableRow>
  );
}

function CreatedKeyDialog({
  createdKey,
  onDismiss,
}: {
  createdKey: APIKeyCreatedResponse | undefined;
  onDismiss: () => void;
}) {
  const open = createdKey != null && Boolean(createdKey.raw_key);
  return (
    <Dialog onOpenChange={(next) => !next && onDismiss()} open={open}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Copy API key now</DialogTitle>
        </DialogHeader>
        <p>This secret is shown once. Store it in your secret manager before closing.</p>
        <Input readOnly value={createdKey?.raw_key ?? ''} />
        <DialogFooter>
          <Button
            onClick={async () => {
              if (createdKey?.raw_key) {
                await copyText(createdKey.raw_key);
              }
            }}
            type="button"
            variant="outline"
          >
            <Copy aria-hidden="true" />
            Copy
          </Button>
          <Button onClick={onDismiss} type="button">
            Done
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
