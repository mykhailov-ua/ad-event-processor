import { useState } from 'react';
import { useRunWhenTrue } from '@/hooks/use_run_when_true';

import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import { FilterField } from '@/shell/filter_panel';
import {
  campaignListCellContentClass,
  campaignListEllipsisTextClass,
  campaignListHeaderLabelClass,
  campaignListHeaderShellClass,
  campaignListNameRowMenuSlotClass,
  campaignListTableFullWidthClass,
  campaignListTdClass,
  campaignListThClass,
} from '@/domains/campaigns/list/campaign_list_classes';
import {
  DirectoryTable,
  directoryTableRevalidatingClass,
  TableBody,
  TableHeader,
} from '@/shell/directory_table';
import type { Offer } from '@/api/types';
import { CreativeDirectoryStack } from '@/domains/creative/creative_directory_stack';
import { creativePanelError } from '@/domains/creative/creative_nav';
import { RowActionsMenu } from '@/shell/row_actions_menu';
import { TableHost } from '@/shell/ui_bands';
import { displayTimestamp } from '@/lib/display';
import { cn } from '@/lib/utils';

const OFFER_COLUMN_WIDTHS = {
  name: '22%',
  url: '54%',
  created: '18%',
  actions: '6%',
} as const;

export type OffersDirectoryProps = {
  items?: Offer[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftName?: string;
  draftUrl?: string;
  creating?: boolean;
  createError?: Error;
  createSuccess?: boolean;
  onDraftNameChange?: (value: string) => void;
  onDraftUrlChange?: (value: string) => void;
  onCreateOffer?: () => void;
  editingOffer?: Offer | null;
  editName?: string;
  editUrl?: string;
  saving?: boolean;
  editError?: Error;
  onOpenEditOffer?: (offer: Offer) => void;
  onCloseEditOffer?: () => void;
  onEditNameChange?: (value: string) => void;
  onEditUrlChange?: (value: string) => void;
  onSaveOffer?: () => void;
  actingOfferId?: string | null;
  actionError?: Error;
  onDeleteOffer?: (offer: Offer) => void;
};

export function OffersDirectory({
  items,
  fetching,
  error,
  hasSnapshot,
  draftName = '',
  draftUrl = '',
  creating = false,
  createError,
  createSuccess = false,
  onDraftNameChange,
  onDraftUrlChange,
  onCreateOffer,
  editingOffer = null,
  editName = '',
  editUrl = '',
  saving = false,
  editError,
  onOpenEditOffer,
  onCloseEditOffer,
  onEditNameChange,
  onEditUrlChange,
  onSaveOffer,
  actingOfferId = null,
  actionError,
  onDeleteOffer,
}: OffersDirectoryProps) {
  const [createOpen, setCreateOpen] = useState(false);
  const editOpen = editingOffer != null;
  const acting = actingOfferId != null || saving;

  useRunWhenTrue(createSuccess, () => setCreateOpen(false));

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={4} />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Offers">
        <CreativeDirectoryStack>
          {creativePanelError(error, 'Could not load offers')}
        </CreativeDirectoryStack>
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="Offers"
      actions={
        onCreateOffer ? (
          <Button onClick={() => setCreateOpen(true)} type="button">
            Create offer
          </Button>
        ) : undefined
      }
    >
      <CreativeDirectoryStack>
      {onCreateOffer ? (
        <Dialog onOpenChange={setCreateOpen} open={createOpen}>
          <DialogContent className="max-w-lg">
            <DialogHeader>
              <DialogTitle>Create offer</DialogTitle>
            </DialogHeader>
            <div className="grid gap-4">
              <FilterField htmlFor="offer-create-name" label="Name">
                <Input
                  id="offer-create-name"
                  placeholder="Offer name..."
                  value={draftName}
                  onChange={(event) => onDraftNameChange?.(event.target.value)}
                />
              </FilterField>
              <FilterField htmlFor="offer-create-url" label="URL">
                <Input
                  id="offer-create-url"
                  placeholder="https://..."
                  value={draftUrl}
                  onChange={(event) => onDraftUrlChange?.(event.target.value)}
                />
              </FilterField>
              {createError ? creativePanelError(createError, 'Could not create offer') : null}
            </div>
            <DialogFooter>
              <PrimaryActionButton loading={creating} onClick={onCreateOffer} type="button">
                Create offer
              </PrimaryActionButton>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}

      {onSaveOffer && onCloseEditOffer ? (
        <Dialog
          onOpenChange={(open) => {
            if (!open) {
              onCloseEditOffer();
            }
          }}
          open={editOpen}
        >
          <DialogContent className="max-w-lg">
            <DialogHeader>
              <DialogTitle>Edit offer</DialogTitle>
            </DialogHeader>
            <div className="grid gap-4">
              <FilterField htmlFor="offer-edit-name" label="Name">
                <Input
                  id="offer-edit-name"
                  value={editName}
                  onChange={(event) => onEditNameChange?.(event.target.value)}
                />
              </FilterField>
              <FilterField htmlFor="offer-edit-url" label="URL">
                <Input
                  id="offer-edit-url"
                  value={editUrl}
                  onChange={(event) => onEditUrlChange?.(event.target.value)}
                />
              </FilterField>
              {editError ? creativePanelError(editError, 'Could not update offer') : null}
            </div>
            <DialogFooter>
              <SecondaryActionButton onClick={onCloseEditOffer} type="button">
                Cancel
              </SecondaryActionButton>
              <PrimaryActionButton loading={saving} onClick={onSaveOffer} type="button">
                Save offer
              </PrimaryActionButton>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}

      <div aria-atomic="true" aria-live="polite">
        {(items ?? []).length === 0 ? (
          <EmptyState
            variant="blank-slate"
            title="No offers"
            description="Create an offer to link landers and flows."
            actionLabel={onCreateOffer ? 'Create offer' : undefined}
            onAction={onCreateOffer ? () => setCreateOpen(true) : undefined}
          />
        ) : (
          <TableHost className="w-full">
            <DirectoryTable
              className={cn(
                'w-full rounded-none border-0 shadow-none',
                directoryTableRevalidatingClass(fetching && hasSnapshot)
              )}
              fixedLayout
              nested
              tableClassName={campaignListTableFullWidthClass}
              tableStyle={{
                width: '100%',
                tableLayout: 'fixed',
              }}
            >
              <colgroup>
                <col style={{ width: OFFER_COLUMN_WIDTHS.name }} />
                <col style={{ width: OFFER_COLUMN_WIDTHS.url }} />
                <col style={{ width: OFFER_COLUMN_WIDTHS.created }} />
                <col style={{ width: OFFER_COLUMN_WIDTHS.actions }} />
              </colgroup>
              <TableHeader>
                <tr>
                  <th className={campaignListThClass}>
                    <div className={campaignListHeaderShellClass}>
                      <span className={campaignListHeaderLabelClass}>Name</span>
                    </div>
                  </th>
                  <th className={campaignListThClass}>
                    <div className={campaignListHeaderShellClass}>
                      <span className={campaignListHeaderLabelClass}>URL</span>
                    </div>
                  </th>
                  <th className={campaignListThClass}>
                    <div className={campaignListHeaderShellClass}>
                      <span className={campaignListHeaderLabelClass}>Created</span>
                    </div>
                  </th>
                  <th className={campaignListThClass}>
                    <div className={campaignListHeaderShellClass}>
                      <span className={campaignListHeaderLabelClass} />
                    </div>
                  </th>
                </tr>
              </TableHeader>
              <TableBody>
                {(items ?? []).map((row) => (
                  <tr key={row.id}>
                    <td className={campaignListTdClass}>
                      <span className={campaignListCellContentClass} title={row.name}>
                        {row.name}
                      </span>
                    </td>
                    <td className={campaignListTdClass}>
                      <span className={campaignListEllipsisTextClass} title={row.url}>
                        {row.url}
                      </span>
                    </td>
                    <td className={campaignListTdClass}>
                      <span
                        className={campaignListCellContentClass}
                        title={displayTimestamp(row.created_at)}
                      >
                        {displayTimestamp(row.created_at)}
                      </span>
                    </td>
                    <td className={campaignListTdClass}>
                      <div className={campaignListNameRowMenuSlotClass}>
                        {onOpenEditOffer || onDeleteOffer ? (
                          <RowActionsMenu
                            ariaLabel={`Actions for ${row.name}`}
                            disabled={acting && actingOfferId === row.id}
                          >
                            {onOpenEditOffer ? (
                              <DropdownMenuItem onClick={() => onOpenEditOffer(row)}>
                                Edit
                              </DropdownMenuItem>
                            ) : null}
                            {onDeleteOffer ? (
                              <DropdownMenuItem onClick={() => onDeleteOffer(row)}>
                                Delete
                              </DropdownMenuItem>
                            ) : null}
                          </RowActionsMenu>
                        ) : null}
                      </div>
                    </td>
                  </tr>
                ))}
              </TableBody>
            </DirectoryTable>
          </TableHost>
        )}
      </div>

      {actionError ? creativePanelError(actionError, 'Offer action failed') : null}
      {error && hasSnapshot ? creativePanelError(error, 'Refresh failed') : null}
      </CreativeDirectoryStack>
    </PageChrome>
  );
}
