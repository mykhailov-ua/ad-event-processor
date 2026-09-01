import { useEffect, useState } from 'react';

import { PrimaryActionButton, SecondaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { CustomerScopeBar } from '@/components/system/customer_scope_bar';
import { EmptyState } from '@/components/system/empty_state';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { RowActionsMenu } from '@/components/system/row_actions_menu';
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
import { Label } from '@/components/ui/label';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { SavedView } from '@/api/types';
import { PortalsNav, portalsPanelError } from '@/domains/portals/portals_nav';
import { displayTimestamp } from '@/lib/display';

export type SavedViewsPanelProps = {
  views: SavedView[];
  appliedCustomerId: string;
  draftCustomerId: string;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  draftName: string;
  draftReportKey: string;
  draftSpecJson: string;
  editRows: Record<string, { name: string; report_key: string; spec_json: string }>;
  acting: boolean;
  actionError: Error | undefined;
  onDraftCustomerIdChange: (value: string) => void;
  onApplyCustomerScope: () => void;
  onDraftNameChange: (value: string) => void;
  onDraftReportKeyChange: (value: string) => void;
  onDraftSpecJsonChange: (value: string) => void;
  onEditRowChange: (
    id: string,
    field: 'name' | 'report_key' | 'spec_json',
    value: string,
  ) => void;
  onCreateView: () => void;
  onUpdateView: (id: string) => void;
  onDeleteView: (id: string) => void;
  createSuccess?: boolean;
};

export function SavedViewsPanel({
  views,
  appliedCustomerId,
  draftCustomerId,
  fetching,
  error,
  hasSnapshot,
  draftName,
  draftReportKey,
  draftSpecJson,
  editRows,
  acting,
  actionError,
  onDraftCustomerIdChange,
  onApplyCustomerScope,
  onDraftNameChange,
  onDraftReportKeyChange,
  onDraftSpecJsonChange,
  onEditRowChange,
  onCreateView,
  onUpdateView,
  onDeleteView,
  createSuccess = false,
}: SavedViewsPanelProps) {
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteViewId, setDeleteViewId] = useState<string | undefined>();

  useEffect(() => {
    if (createSuccess) {
      setCreateOpen(false);
    }
  }, [createSuccess]);

  if (!appliedCustomerId) {
    return (
      <PageChrome title="Saved views">
        <PortalsNav />
        <CustomerScopeBar
          appliedCustomerId={appliedCustomerId}
          draftCustomerId={draftCustomerId}
          onApply={onApplyCustomerScope}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
        />
        <EmptyState
          title="Customer required"
          description="Apply a customer ID to list saved report views."
        />
      </PageChrome>
    );
  }

  return (
    <PageChrome
      title="Saved views"
      actions={
        <Button className="h-9 text-sm" onClick={() => setCreateOpen(true)} type="button">
          Create saved view
        </Button>
      }
    >
      <PortalsNav />

      <CustomerScopeBar
        appliedCustomerId={appliedCustomerId}
        draftCustomerId={draftCustomerId}
        onApply={onApplyCustomerScope}
        onDraftCustomerIdChange={onDraftCustomerIdChange}
      />

      <Dialog onOpenChange={setCreateOpen} open={createOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Create saved view</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="view-name">Name</Label>
              <Input
                id="view-name"
                className="h-9 text-sm"
                value={draftName}
                onChange={(event) => onDraftNameChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="view-report-key">Report key</Label>
              <Input
                id="view-report-key"
                className="h-9 text-sm"
                value={draftReportKey}
                onChange={(event) => onDraftReportKeyChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="view-spec">Spec JSON (optional)</Label>
              <Input
                id="view-spec"
                className="h-9 text-sm"
                placeholder="{}"
                value={draftSpecJson}
                onChange={(event) => onDraftSpecJsonChange(event.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <PrimaryActionButton disabled={acting} loading={acting} onClick={onCreateView} type="button">
              Create
            </PrimaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        onOpenChange={(open) => {
          if (!open) {
            setDeleteViewId(undefined);
          }
        }}
        open={Boolean(deleteViewId)}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Delete saved view</DialogTitle>
          </DialogHeader>
          <DialogFooter className="gap-2 sm:gap-0">
            <SecondaryActionButton onClick={() => setDeleteViewId(undefined)} type="button">
              Cancel
            </SecondaryActionButton>
            <Button
              className="h-9 text-sm"
              disabled={acting}
              loading={acting}
              onClick={() => {
                if (deleteViewId) {
                  onDeleteView(deleteViewId);
                  setDeleteViewId(undefined);
                }
              }}
              shape="pill"
              type="button"
              variant="destructive"
            >
              Delete view
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {fetching && !hasSnapshot && !error ? (
        <PageSkeleton />
      ) : error && !hasSnapshot ? (
        portalsPanelError(error, 'Could not load saved views')
      ) : (
        <div aria-atomic="true" aria-live="polite">
          {views.length === 0 ? (
            <EmptyState
              variant="blank-slate"
              title="No saved views"
              description="Save a report filter spec for quick reuse."
              actionLabel="Create saved view"
              onAction={() => setCreateOpen(true)}
            />
          ) : (
            <div className="ui-table-frame">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Report</TableHead>
                    <TableHead>Updated</TableHead>
                    <TableHead />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {views.map((row) => {
                    const id = row.id ?? '';
                    const edit = editRows[id] ?? {
                      name: row.name ?? '',
                      report_key: row.report_key ?? '',
                      spec_json: '',
                    };
                    return (
                      <TableRow key={id || row.name}>
                        <TableCell>
                          {id ? (
                            <Input
                              className="h-9 text-sm"
                              value={edit.name}
                              onChange={(event) => onEditRowChange(id, 'name', event.target.value)}
                            />
                          ) : (
                            row.name ?? ''
                          )}
                        </TableCell>
                        <TableCell>
                          {id ? (
                            <Input
                              className="h-9 font-mono text-xs"
                              value={edit.report_key}
                              onChange={(event) =>
                                onEditRowChange(id, 'report_key', event.target.value)
                              }
                            />
                          ) : (
                            <span className="font-mono text-xs">{row.report_key ?? ''}</span>
                          )}
                        </TableCell>
                        <TableCell>{displayTimestamp(row.updated_at)}</TableCell>
                        <TableCell>
                          {id ? (
                            <RowActionsMenu ariaLabel="Saved view actions" disabled={acting}>
                              <DropdownMenuItem disabled={acting} onClick={() => onUpdateView(id)}>
                                Save
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                className="text-destructive focus:text-destructive"
                                disabled={acting}
                                onClick={() => setDeleteViewId(id)}
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
              </Table>
            </div>
          )}
        </div>
      )}

      {actionError ? portalsPanelError(actionError, 'Saved view action failed') : null}
      {error && hasSnapshot ? portalsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
