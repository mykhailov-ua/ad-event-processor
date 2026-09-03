import { useEffect, useState } from 'react';

import { PrimaryActionButton } from '@/shell/action_buttons';
import { PageChrome } from '@/shell/page_chrome';
import { RowActionsMenu } from '@/shell/row_actions_menu';
import { CampaignScopeBar } from '@/shell/campaign_scope_bar';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import type { TelegramPostback } from '@/api/types';
import { TelegramNav, telegramPanelError } from '@/domains/telegram/telegram_nav';
import { displayTimestamp } from '@/lib/display';

export type TelegramPostbacksDirectoryProps = {
  postbacks: TelegramPostback[];
  appliedCampaignId: string;
  draftCampaignId: string;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftPostbackUrl: string;
  editUrls: Record<string, string>;
  acting: boolean;
  actionError: Error | undefined;
  actionMessage?: string;
  createSuccess?: boolean;
  onDraftCampaignIdChange: (value: string) => void;
  onApplyCampaignScope: () => void;
  onDraftPostbackUrlChange: (value: string) => void;
  onEditUrlChange: (id: string, value: string) => void;
  onCreatePostback: () => void;
  onUpdatePostback: (id: string) => void;
  onDeletePostback: (id: string) => void;
  onTestPostback: (id: string) => void;
};

export function TelegramPostbacksDirectory({
  postbacks,
  appliedCampaignId,
  draftCampaignId,
  fetching,
  error,
  hasSnapshot,
  draftPostbackUrl,
  editUrls,
  acting,
  actionError,
  actionMessage,
  createSuccess = false,
  onDraftCampaignIdChange,
  onApplyCampaignScope,
  onDraftPostbackUrlChange,
  onEditUrlChange,
  onCreatePostback,
  onUpdatePostback,
  onDeletePostback,
  onTestPostback,
}: TelegramPostbacksDirectoryProps) {
  const [createOpen, setCreateOpen] = useState(false);

  useEffect(() => {
    if (createSuccess) {
      setCreateOpen(false);
    }
  }, [createSuccess]);

  if (!appliedCampaignId) {
    return (
      <PageChrome title="Telegram postbacks">
        <TelegramNav />
        <CampaignScopeBar
          appliedCampaignId={appliedCampaignId}
          draftCampaignId={draftCampaignId}
          onApply={onApplyCampaignScope}
          onDraftCampaignIdChange={onDraftCampaignIdChange}
        />
        <EmptyState
          title="Campaign required"
          description="Apply a campaign ID to list and manage Telegram postbacks."
        />
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="Telegram postbacks"
      actions={
        <PrimaryActionButton onClick={() => setCreateOpen(true)} type="button">
          Create postback
        </PrimaryActionButton>
      }
    >
      <TelegramNav />

      <CampaignScopeBar
        appliedCampaignId={appliedCampaignId}
        draftCampaignId={draftCampaignId}
        onApply={onApplyCampaignScope}
        onDraftCampaignIdChange={onDraftCampaignIdChange}
      />

      <Dialog onOpenChange={setCreateOpen} open={createOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Create postback</DialogTitle>
          </DialogHeader>
          <div className="grid gap-2">
            <Label htmlFor="telegram-postback-url">Postback URL</Label>
            <Input
              id="telegram-postback-url"
              value={draftPostbackUrl}
              onChange={(event) => onDraftPostbackUrlChange(event.target.value)}
            />
          </div>
          <DialogFooter>
            <PrimaryActionButton loading={acting} onClick={onCreatePostback} type="button">
              Create
            </PrimaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {fetching && !hasSnapshot && !error ? (
        <PageSkeleton />
      ) : error && !hasSnapshot ? (
        telegramPanelError(error, 'Could not load postbacks')
      ) : postbacks.length === 0 ? (
        <EmptyState title="No postbacks" description="No Telegram postback URLs for this campaign." />
      ) : (
        <DirectoryTable>
            <TableHeader>
              <TableRow>
                <DirectoryTableHead>URL</DirectoryTableHead>
                <DirectoryTableHead>Updated</DirectoryTableHead>
                <DirectoryTableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {postbacks.map((row) => {
                const id = row.id ?? '';
                return (
                  <TableRow key={id || row.postback_url}>
                    <TableCell>
                      {id ? (
                        <Input
                          className="font-mono text-xs"
                          value={editUrls[id] ?? row.postback_url ?? ''}
                          onChange={(event) => onEditUrlChange(id, event.target.value)}
                        />
                      ) : (
                        <span className="font-mono text-xs">{row.postback_url ?? ''}</span>
                      )}
                    </TableCell>
                    <TableCell>{displayTimestamp(row.updated_at)}</TableCell>
                    <TableCell>
                      {id ? (
                        <RowActionsMenu ariaLabel="Postback actions" disabled={acting}>
                          <DropdownMenuItem disabled={acting} onClick={() => onUpdatePostback(id)}>
                            Save
                          </DropdownMenuItem>
                          <DropdownMenuItem disabled={acting} onClick={() => onTestPostback(id)}>
                            Test
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            className="text-destructive focus:text-destructive"
                            disabled={acting}
                            onClick={() => onDeletePostback(id)}
                          >
                            Delete
                          </DropdownMenuItem>
                        </RowActionsMenu>
                      ) : null}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </DirectoryTable>
      )}

      {actionMessage ? (
        <p className="text-sm text-muted-foreground">{actionMessage}</p>
      ) : null}
      {actionError ? telegramPanelError(actionError, 'Postback action failed') : null}
      {error && hasSnapshot ? telegramPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
