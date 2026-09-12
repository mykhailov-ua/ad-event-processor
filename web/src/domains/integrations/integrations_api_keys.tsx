import { useMemo, useState } from 'react';
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
import { FieldGroup } from '@/shell/field_group';
import type { DirectoryOverviewField } from '@/shell/directory_overview_dialog';
import {
  DirectorySelectOverviewTable,
  directoryRecordMap,
  directoryOperateRows,
} from '@/shell/directory_select_overview_table';
import { DirectoryRowActionsMenu } from '@/shell/directory_row_actions_menu';
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import { TableHost } from '@/shell/ui_bands';
import {
  IntegrationsPageWithLoad,
  integrationsPanelError,
} from '@/domains/integrations/integrations_nav';
import {
  SELF_SERVE_API_KEY_SCOPES,
  type useIntegrationsApiKeysPageWorkspace,
} from '@/domains/integrations/use_integrations_api_keys_page_workspace';
import { ValidationErrorBlock } from '@/shell/validation_error_block';
import { adminTypography } from '@/lib/admin_kit';

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

function buildApiKeyOverviewFields(row: APIKeySummary): DirectoryOverviewField[] {
  const scopes = row.scopes ?? [];
  return [
    { label: 'Name', value: row.name },
    {
      label: 'Scopes',
      value:
        scopes.length === 0
          ? ''
          : scopes.map((scope) => (
              <Badge key={scope} variant="secondary">
                {scope}
              </Badge>
            )),
    },
    { label: 'Created', value: displayTimestamp(row.created_at) },
    {
      label: 'Expires',
      value: row.expires_at ? displayTimestamp(row.expires_at) : '',
    },
    { label: 'ID', value: <span className={adminTypography.monoData}>{row.id}</span> },
  ];
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
  formValidationError,
}: IntegrationsApiKeysProps) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const actionsDisabled = fetching || revokingId != null;

  const recordById = useMemo(() => directoryRecordMap(keys, (row) => row.id), [keys]);
  const rows = useMemo(
    () =>
      directoryOperateRows(
        keys,
        (row) => row.id,
        (row) => row.name
      ),
    [keys]
  );

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
        <FieldGroup disabled={creating} legend="Scopes">
          {SELF_SERVE_API_KEY_SCOPES.map((scope) => (
            <label key={scope}>
              <Checkbox
                checked={draftScopes.includes(scope)}
                disabled={creating}
                onCheckedChange={(checked) => onToggleScope(scope, checked === true)}
              />
              {scope}
            </label>
          ))}
        </FieldGroup>
        {formValidationError ? (
          <ValidationErrorBlock error={formValidationError} title="Check key fields" />
        ) : null}
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
          <TableHost>
            <DirectorySelectOverviewTable
              buildOverviewFields={buildApiKeyOverviewFields}
              disabled={actionsDisabled}
              overviewTitle={(row) => row.name}
              recordById={recordById}
              revalidating={listRevalidating}
              renderActions={(tableRow, record, openOverview) => {
                const revoking = revokingId === record.id;
                return (
                  <DirectoryRowActionsMenu
                    ariaLabel={`Actions for ${String(tableRow.label)}`}
                    disabled={actionsDisabled || !record.id}
                    onOverview={openOverview}
                  >
                    <DropdownMenuItem
                      disabled={actionsDisabled || revoking || !record.id}
                      onClick={() => onRevoke(record)}
                    >
                      {revoking ? 'Revoking...' : 'Revoke'}
                    </DropdownMenuItem>
                  </DirectoryRowActionsMenu>
                );
              }}
              rows={rows}
              selectedId={selectedId}
              onSelectedIdChange={setSelectedId}
            />
          </TableHost>
        )}
      </section>

      <CreatedKeyDialog createdKey={createdKey} onDismiss={onDismissCreatedKey} />
    </IntegrationsPageWithLoad>
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
