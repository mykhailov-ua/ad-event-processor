import { useEffect, useState } from 'react';

import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { PaginationPrevNext } from '@/shell/pagination_prev_next';
import { RowActionsMenu } from '@/shell/row_actions_menu';
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
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { TeamBudgetApproval, TeamMember, TeamOverview } from '@/api/types';
import { displayTimestamp } from '@/lib/display';

export type TeamMemberEditDraft = {
  role: string;
  is_blocked: boolean;
  spend_cap_micro: string;
};

export type TeamRosterTab = 'members' | 'approvals';

const ROSTER_TABS: { id: TeamRosterTab; label: string }[] = [
  { id: 'members', label: 'Members' },
  { id: 'approvals', label: 'Budget approvals' },
];

export type TeamOverviewViewProps = {
  rosterTab: TeamRosterTab;
  onRosterTabChange: (tab: TeamRosterTab) => void;
  overview: TeamOverview | undefined;
  members: TeamMember[];
  membersTotal: number;
  membersLimit: number;
  membersOffset: number;
  membersCustomerId: string;
  approvals: TeamBudgetApproval[];
  approvalsTotal: number;
  approvalsLimit: number;
  approvalsOffset: number;
  approvalsCustomerId: string;
  draftCustomerId: string;
  draftInviteEmail: string;
  draftInviteRole: string;
  memberDrafts: Record<string, TeamMemberEditDraft>;
  fetching: boolean;
  membersFetching: boolean;
  approvalsFetching: boolean;
  inviting: boolean;
  error: Error | undefined;
  membersError: Error | undefined;
  approvalsError: Error | undefined;
  actionError: Error | undefined;
  inviteSuccess: boolean;
  hasSnapshot: boolean;
  hasMembersSnapshot: boolean;
  hasApprovalsSnapshot: boolean;
  actingId?: string;
  memberUpdatingId?: string;
  onDraftCustomerIdChange: (value: string) => void;
  onDraftInviteEmailChange: (value: string) => void;
  onDraftInviteRoleChange: (value: string) => void;
  onMemberDraftChange: (memberId: string, patch: Partial<TeamMemberEditDraft>) => void;
  onApplyCustomer: () => void;
  onInvite: () => void;
  onMembersPageChange: (nextOffset: number) => void;
  onApprovalsPageChange: (nextOffset: number) => void;
  onSaveMember: (memberId: string) => void;
  onApprove: (id: string) => void;
  onDeny: (id: string) => void;
};

function memberDraftFromRow(member: TeamMember): TeamMemberEditDraft {
  return {
    role: member.role ?? '',
    is_blocked: member.is_blocked ?? false,
    spend_cap_micro: member.spend_cap_micro != null ? String(member.spend_cap_micro) : '',
  };
}

export function TeamOverviewView({
  rosterTab,
  onRosterTabChange,
  overview,
  members,
  membersTotal,
  membersLimit,
  membersOffset,
  membersCustomerId,
  approvals,
  approvalsTotal,
  approvalsLimit,
  approvalsOffset,
  approvalsCustomerId,
  draftCustomerId,
  draftInviteEmail,
  draftInviteRole,
  memberDrafts,
  fetching,
  membersFetching,
  approvalsFetching,
  inviting,
  error,
  membersError,
  approvalsError,
  actionError,
  inviteSuccess,
  hasSnapshot,
  hasMembersSnapshot,
  hasApprovalsSnapshot,
  actingId,
  memberUpdatingId,
  onDraftCustomerIdChange,
  onDraftInviteEmailChange,
  onDraftInviteRoleChange,
  onMemberDraftChange,
  onApplyCustomer,
  onInvite,
  onMembersPageChange,
  onApprovalsPageChange,
  onSaveMember,
  onApprove,
  onDeny,
}: TeamOverviewViewProps) {
  const [inviteOpen, setInviteOpen] = useState(false);

  useEffect(() => {
    if (inviteSuccess) {
      setInviteOpen(false);
    }
  }, [inviteSuccess]);

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={5} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load team overview" message={error.message} />;
  }

  const membersList = members;

  return (
    <PageChrome
      title="Team"
      actions={
        <PrimaryActionButton
          disabled={!draftCustomerId.trim()}
          onClick={() => setInviteOpen(true)}
          type="button"
        >
          Invite member
        </PrimaryActionButton>
      }
    >
      <form
        className="grid max-w-md grid-cols-[1fr_auto] items-end gap-4"
        onSubmit={(event) => {
          event.preventDefault();
          onApplyCustomer();
        }}
      >
        <div className="grid gap-2">
          <Label htmlFor="team-customer-id">Customer ID</Label>
          <Input
            id="team-customer-id"
            className="text-sm"
            value={draftCustomerId}
            onChange={(event) => onDraftCustomerIdChange(event.target.value)}
          />
        </div>
        <SecondaryActionButton type="submit">Load</SecondaryActionButton>
      </form>

      {overview ? (
        <div className="ui-filter-panel gap-2 text-sm">
          <div className="flex flex-wrap gap-4">
            <span>{overview.customer_name ?? overview.customer_id}</span>
            {overview.cost_center ? <span>Cost center: {overview.cost_center}</span> : null}
            {overview.balance_micro != null ? (
              <span>
                Balance: {overview.balance_micro} {overview.currency ?? 'micro'}
              </span>
            ) : null}
          </div>
          {overview.license ? (
            <div className="flex flex-wrap items-center gap-2 text-muted-foreground">
              <span>License: {overview.license.state ?? ''}</span>
              {overview.license.plan_code ? (
                <Badge variant="outline">{overview.license.plan_code}</Badge>
              ) : null}
            </div>
          ) : null}
        </div>
      ) : null}

      <Dialog onOpenChange={setInviteOpen} open={inviteOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Invite member</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 md:grid-cols-[repeat(auto-fill,minmax(12rem,1fr))]">
            <div className="grid gap-2 md:col-span-2">
              <Label htmlFor="team-invite-email">Email</Label>
              <Input
                id="team-invite-email"
                type="email"
                value={draftInviteEmail}
                onChange={(event) => onDraftInviteEmailChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="team-invite-role">Role</Label>
              <Input
                id="team-invite-role"
                value={draftInviteRole}
                onChange={(event) => onDraftInviteRoleChange(event.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <PrimaryActionButton
              disabled={
                !draftCustomerId.trim() ||
                !draftInviteEmail.trim() ||
                !draftInviteRole.trim()
              }
              loading={inviting}
              onClick={onInvite}
              type="button"
            >
              Send invite
            </PrimaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Tabs
        onValueChange={(value) => onRosterTabChange(value as TeamRosterTab)}
        value={rosterTab}
      >
        <TabsList>
          {ROSTER_TABS.map((item) => (
            <TabsTrigger key={item.id} value={item.id}>
              {item.label}
            </TabsTrigger>
          ))}
        </TabsList>

        <TabsContent className="grid gap-6" value="members">
      <h2 className="text-base font-semibold">Members</h2>
      {membersFetching && !hasMembersSnapshot ? (
        <p className="text-sm text-muted-foreground">Loading members...</p>
      ) : !membersCustomerId ? (
        <EmptyState title="Customer required" description="Load a customer to review team members." />
      ) : membersList.length === 0 ? (
        <EmptyState title="No members" description="Team roster is empty for this customer." />
      ) : (
        <>
          <form
            className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4"
            onSubmit={(event) => event.preventDefault()}
          >
            <PaginationPrevNext
              canGoPrev={membersOffset > 0}
              canGoNext={membersOffset + membersList.length < membersTotal}
              disabled={membersFetching}
              onPrev={() => onMembersPageChange(Math.max(0, membersOffset - membersLimit))}
              onNext={() => onMembersPageChange(membersOffset + membersLimit)}
            />
          </form>
          <div className="ui-table-frame">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Email</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>Campaigns</TableHead>
                  <TableHead>Spend cap</TableHead>
                  <TableHead>Blocked</TableHead>
                  <TableHead>Joined</TableHead>
                  <TableHead />
                </TableRow>
              </TableHeader>
              <TableBody>
                {membersList.map((member) => {
                  const memberId = member.user_id ?? '';
                  const draft = memberDrafts[memberId] ?? memberDraftFromRow(member);
                  const updating = memberUpdatingId === memberId;
                  return (
                    <TableRow key={memberId || member.email}>
                      <TableCell>{member.email ?? ''}</TableCell>
                      <TableCell>
                        <Input
                          aria-label={`Role for ${member.email ?? memberId}`}
                          className="min-w-[5rem]"
                          value={draft.role}
                          onChange={(event) =>
                            onMemberDraftChange(memberId, { role: event.target.value })
                          }
                        />
                      </TableCell>
                      <TableCell className="tabular-nums">{member.campaigns_owned ?? ''}</TableCell>
                      <TableCell>
                        <Input
                          aria-label={`Spend cap for ${member.email ?? memberId}`}
                          className="min-w-[6rem] font-mono text-xs"
                          inputMode="numeric"
                          value={draft.spend_cap_micro}
                          onChange={(event) =>
                            onMemberDraftChange(memberId, { spend_cap_micro: event.target.value })
                          }
                        />
                      </TableCell>
                      <TableCell>
                        <Checkbox
                          aria-label={`Blocked for ${member.email ?? memberId}`}
                          checked={draft.is_blocked}
                          onCheckedChange={(checked) =>
                            onMemberDraftChange(memberId, { is_blocked: checked === true })
                          }
                        />
                      </TableCell>
                      <TableCell>
                        {displayTimestamp(member.created_at, member.created_at_display)}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          disabled={!memberId || updating}
                          onClick={() => onSaveMember(memberId)}
                         
                          type="button"
                          variant="outline"
                        >
                          {updating ? 'Saving...' : 'Save'}
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
        </>
      )}

      {membersError ? (
        <ErrorBlock title="Could not load members" message={membersError.message} />
      ) : null}
        </TabsContent>

        <TabsContent className="grid gap-6" value="approvals">
      <h2 className="text-base font-semibold">Budget approvals</h2>
      {approvalsFetching && !hasApprovalsSnapshot ? (
        <p className="text-sm text-muted-foreground">Loading approvals...</p>
      ) : !approvalsCustomerId ? (
        <EmptyState title="Customer required" description="Load a customer to review budget approvals." />
      ) : approvals.length === 0 ? (
        <EmptyState title="No pending approvals" description="Budget approval queue is empty." />
      ) : (
        <>
          <form
            className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4"
            onSubmit={(event) => event.preventDefault()}
          >
            <PaginationPrevNext
              canGoPrev={approvalsOffset > 0}
              canGoNext={approvalsOffset + approvals.length < approvalsTotal}
              disabled={approvalsFetching}
              onPrev={() => onApprovalsPageChange(Math.max(0, approvalsOffset - approvalsLimit))}
              onNext={() => onApprovalsPageChange(approvalsOffset + approvalsLimit)}
            />
          </form>
          <div className="ui-table-frame">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Status</TableHead>
                  <TableHead>User</TableHead>
                  <TableHead>Campaign</TableHead>
                  <TableHead>Requested</TableHead>
                  <TableHead>Previous</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead />
                </TableRow>
              </TableHeader>
              <TableBody>
                {approvals.map((row) => {
                  const rowId = row.id ?? '';
                  return (
                    <TableRow key={rowId}>
                      <TableCell>{row.status ?? ''}</TableCell>
                      <TableCell className="font-mono text-xs">{row.user_id ?? ''}</TableCell>
                      <TableCell className="font-mono text-xs">{row.campaign_id ?? ''}</TableCell>
                      <TableCell className="tabular-nums">{row.requested_budget_micro ?? ''}</TableCell>
                      <TableCell className="tabular-nums">{row.previous_budget_micro ?? ''}</TableCell>
                      <TableCell>
                        {displayTimestamp(row.created_at, row.created_at_display)}
                      </TableCell>
                      <TableCell className="text-right">
                        <RowActionsMenu
                          ariaLabel="Approval actions"
                          disabled={!rowId || actingId === rowId}
                        >
                          <DropdownMenuItem
                            disabled={!rowId || actingId === rowId}
                            onClick={() => onApprove(rowId)}
                          >
                            Approve
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            className="text-destructive focus:text-destructive"
                            disabled={!rowId || actingId === rowId}
                            onClick={() => onDeny(rowId)}
                          >
                            Deny
                          </DropdownMenuItem>
                        </RowActionsMenu>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>
        </>
      )}

      {approvalsError ? (
        <ErrorBlock title="Could not load approvals" message={approvalsError.message} />
      ) : null}
        </TabsContent>
      </Tabs>
      {actionError ? <ErrorBlock title="Action failed" message={actionError.message} /> : null}
      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageChrome>
  );
}
