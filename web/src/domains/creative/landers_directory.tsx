import { useState } from 'react';
import { useRunWhenTrue } from '@/hooks/use_run_when_true';

import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { PageLayout } from '@/shell/page_layout';
import { PageSkeleton } from '@/shell/page_skeleton';
import { ErrorBlock } from '@/shell/error_block';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { Lander } from '@/api/types';
import { creativePanelError } from '@/domains/creative/creative_nav';
import { LandersListTable } from '@/domains/creative/landers_list_table';
import { LandersListToolbar } from '@/domains/creative/landers_list_toolbar';
import { useLandersListView } from '@/domains/creative/use_landers_list_view';

export type LandersDirectoryProps = {
  items?: Lander[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  listLastUpdatedAt?: string | null;
  onRefresh?: () => void;
  draftName?: string;
  draftUrl?: string;
  creating?: boolean;
  createError?: Error;
  createSuccess?: boolean;
  onDraftNameChange?: (value: string) => void;
  onDraftUrlChange?: (value: string) => void;
  onCreateLander?: () => void;
  editingLander?: Lander | null;
  editName?: string;
  editUrl?: string;
  saving?: boolean;
  editError?: Error;
  onOpenEditLander?: (lander: Lander) => void;
  onCloseEditLander?: () => void;
  onEditNameChange?: (value: string) => void;
  onEditUrlChange?: (value: string) => void;
  onSaveLander?: () => void;
  actingLanderId?: string | null;
  actionError?: Error;
  onDeleteLander?: (lander: Lander) => void;
  onCopyUrl?: (url: string) => void;
};

export function LandersDirectory({
  items,
  fetching,
  error,
  hasSnapshot,
  listLastUpdatedAt = null,
  onRefresh,
  draftName = '',
  draftUrl = '',
  creating = false,
  createError,
  createSuccess = false,
  onDraftNameChange,
  onDraftUrlChange,
  onCreateLander,
  editingLander = null,
  editName = '',
  editUrl = '',
  saving = false,
  editError,
  onOpenEditLander,
  onCloseEditLander,
  onEditNameChange,
  onEditUrlChange,
  onSaveLander,
  actingLanderId = null,
  actionError,
  onDeleteLander,
  onCopyUrl,
}: LandersDirectoryProps) {
  const [createOpen, setCreateOpen] = useState(false);
  const editOpen = editingLander != null;
  const acting = actingLanderId != null || saving;

  const listView = useLandersListView(items);

  useRunWhenTrue(createSuccess, () => setCreateOpen(false));

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={5} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock message={error.message} title="Could not load landers" />;
  }

  const emptyMessage =
    listView.filtersActive || listView.hostingFilter
      ? 'No landers match the current filters.'
      : 'No landers yet. Create one to attach landing pages to flows.';

  return (
    <>
      <PageLayout
        description="Create landing pages, upload hosted ZIPs, and open the editor to publish."
        mainClassName="min-w-0 w-full"
        title="Landers"
        controlPanel={
          <LandersListToolbar
            acting={acting}
            draftSearch={listView.draftSearch}
            fetching={fetching && hasSnapshot}
            filteredTotal={listView.filteredTotal}
            filtersActive={listView.filtersActive}
            hostingCounts={listView.hostingCounts}
            hostingFilter={listView.hostingFilter}
            listLastUpdatedAt={listLastUpdatedAt}
            onCreateClick={() => setCreateOpen(true)}
            onDraftSearchChange={listView.onDraftSearchChange}
            onHostingFilterChange={listView.onHostingFilterChange}
            onRefresh={onRefresh ?? (() => undefined)}
          />
        }
        footer={
          listView.filteredTotal > 0 ? (
            <DirectoryPaginationFooter
              canGoNext={listView.canGoNext}
              canGoPrev={listView.canGoPrev}
              className="gap-2"
              disabled={fetching}
              limit={listView.limit}
              page={listView.page}
              pageCount={listView.pageCount}
              pageSizeId="landers-page-size"
              rangeLabel={listView.rangeLabel}
              showPrevNext={false}
              onLimitChange={listView.onPageSizeChange}
              onNext={() => listView.onPageChange(listView.offset + listView.limit)}
              onPageChange={(nextPage) =>
                listView.onPageChange((nextPage - 1) * listView.limit)
              }
              onPrev={() => listView.onPageChange(Math.max(0, listView.offset - listView.limit))}
            />
          ) : undefined
        }
      >
        <div className="min-w-0 w-full">
          <LandersListTable
            acting={acting}
            actingLanderId={actingLanderId}
            emptyMessage={emptyMessage}
            fetching={fetching && hasSnapshot}
            items={listView.pageItems}
            onCopyUrl={onCopyUrl}
            onDeleteLander={onDeleteLander}
            onOpenEditLander={onOpenEditLander}
          />
        </div>

        {actionError ? creativePanelError(actionError, 'Lander action failed') : null}
        {error && hasSnapshot ? creativePanelError(error, 'Refresh failed') : null}
      </PageLayout>

      {onCreateLander ? (
        <Dialog onOpenChange={setCreateOpen} open={createOpen}>
          <DialogContent className="max-w-lg">
            <DialogHeader className="gap-1">
              <DialogTitle>Create lander</DialogTitle>
            </DialogHeader>
            <div className="grid gap-4">
              <div className="grid gap-2">
                <Label htmlFor="lander-create-name">Name</Label>
                <Input
                  id="lander-create-name"
                  placeholder="Lander name..."
                  value={draftName}
                  onChange={(event) => onDraftNameChange?.(event.target.value)}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="lander-create-url">URL</Label>
                <Input
                  id="lander-create-url"
                  placeholder="https://..."
                  value={draftUrl}
                  onChange={(event) => onDraftUrlChange?.(event.target.value)}
                />
              </div>
            </div>
            {createError ? creativePanelError(createError, 'Could not create lander') : null}
            <DialogFooter className="justify-end gap-2 sm:flex-row">
              <PrimaryActionButton loading={creating} onClick={onCreateLander} type="button">
                Create lander
              </PrimaryActionButton>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}

      {onSaveLander && onCloseEditLander ? (
        <Dialog
          onOpenChange={(open) => {
            if (!open) {
              onCloseEditLander();
            }
          }}
          open={editOpen}
        >
          <DialogContent className="max-w-lg">
            <DialogHeader className="gap-1">
              <DialogTitle>Edit lander</DialogTitle>
            </DialogHeader>
            <div className="grid gap-4">
              <div className="grid gap-2">
                <Label htmlFor="lander-edit-name">Name</Label>
                <Input
                  id="lander-edit-name"
                  value={editName}
                  onChange={(event) => onEditNameChange?.(event.target.value)}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="lander-edit-url">URL</Label>
                <Input
                  id="lander-edit-url"
                  disabled={Boolean(editingLander?.hosted_asset_id)}
                  placeholder="https://..."
                  value={editUrl}
                  onChange={(event) => onEditUrlChange?.(event.target.value)}
                />
                {editingLander?.hosted_asset_id ? (
                  <p className="text-xs text-muted-foreground">
                    Hosted landers use the published URL. Open the editor to manage files.
                  </p>
                ) : null}
              </div>
            </div>
            {editError ? creativePanelError(editError, 'Could not update lander') : null}
            <DialogFooter className="justify-end gap-2 sm:flex-row">
              <SecondaryActionButton onClick={onCloseEditLander} type="button">
                Cancel
              </SecondaryActionButton>
              <PrimaryActionButton loading={saving} onClick={onSaveLander} type="button">
                Save lander
              </PrimaryActionButton>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}
    </>
  );
}
