import { useState } from 'react';
import { useRunWhenTrue } from '@/hooks/use_run_when_true';

import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import {
  DirectoryFetchError,
  DirectoryMutationError,
  DirectoryPageShell,
} from '@/shell/directory_page_shell';
import { EmptyState } from '@/shell/empty_state';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
  FILTER_PANEL_SUMMARY_CLASS,
  INLINE_FILTER_ACTION_GRID_CLASS,
} from '@/shell/filter_panel';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
  directoryTableRevalidatingClass,
} from '@/shell/directory_table';
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
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import type {
  TeamBudgetApproval,
  TeamMember,
  TeamOverview,
  TeamMetricsResponse,
} from '@/api/types';
import { displayTimestamp } from '@/lib/display';
import { DASHBOARD_RANGE_PRESETS, type DashboardRangePreset } from '@/lib/dashboard_range';
import { TeamMetricsPanel } from '@/domains/team/team_metrics_panel';
import { TeamMyApprovalsPanel } from '@/domains/team/team_my_approvals_panel';

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

export type TeamRoleOption = {
  code: string;
  label: string;
};

export type TeamOverviewViewProps = {
  rosterTab: TeamRosterTab;
  onRosterTabChange: (tab: TeamRosterTab) => void;
  overview: TeamOverview | undefined;
  canViewTeamMetrics: boolean;
  showMyApprovalsPanel: boolean;
  teamMetrics: TeamMetricsResponse | undefined;
  metricsFetching: boolean;
  metricsError: Error | undefined;
  hasMetricsSnapshot: boolean;
  metricsRangeLabel: string;
  draftMetricsFrom: string;
  draftMetricsTo: string;
  draftMetricsPreset: DashboardRangePreset;
  myApprovals: TeamBudgetApproval[];
  myApprovalsFetching: boolean;
  myApprovalsError: Error | undefined;
  hasMyApprovalsSnapshot: boolean;
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
  teamRoleOptions: TeamRoleOption[];
  memberDrafts: Record<string, TeamMemberEditDraft>;
  fetching: boolean;
  membersFetching: boolean;
  membersListRevalidating?: boolean;
  approvalsFetching: boolean;
  approvalsListRevalidating?: boolean;
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
  onDraftMetricsFromChange: (value: string) => void;
  onDraftMetricsToChange: (value: string) => void;
  onDraftMetricsPresetChange: (preset: DashboardRangePreset) => void;
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
  canViewTeamMetrics,
  showMyApprovalsPanel,
  teamMetrics,
  metricsFetching,
  metricsError,
  hasMetricsSnapshot,
  metricsRangeLabel,
  draftMetricsFrom,
  draftMetricsTo,
  draftMetricsPreset,
  myApprovals,
  myApprovalsFetching,
  myApprovalsError,
  hasMyApprovalsSnapshot,
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
  teamRoleOptions,
  memberDrafts,
  fetching,
  membersFetching,
  membersListRevalidating = false,
  approvalsFetching,
  approvalsListRevalidating = false,
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
  onDraftMetricsFromChange,
  onDraftMetricsToChange,
  onDraftMetricsPresetChange,
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

  useRunWhenTrue(inviteSuccess, () => setInviteOpen(false));

  const membersList = members;
  const membersFooterVisible =
    rosterTab === 'members' && Boolean(membersCustomerId) && membersList.length > 0;
  const approvalsFooterVisible =
    rosterTab === 'approvals' && Boolean(approvalsCustomerId) && approvals.length > 0;

  return (
    <DirectoryPageShell
      actions={
        <PrimaryActionButton
          disabled={!draftCustomerId.trim()}
          onClick={() => setInviteOpen(true)}
          type="button"
        >
          Invite member
        </PrimaryActionButton>
      }
      blockingErrorTitle="Could not load team overview"
      controlPanel={
        <FilterPanel>
          <DirectoryFilterForm
            onSubmit={(event) => {
              event.preventDefault();
              onApplyCustomer();
            }}
          >
            <FilterField htmlFor="team-customer-id" label="Customer ID">
              <Input
                id="team-customer-id"
                value={draftCustomerId}
                onChange={(event) => onDraftCustomerIdChange(event.target.value)}
              />
            </FilterField>
            {canViewTeamMetrics ? (
              <>
                <FilterField htmlFor="team-metrics-preset" label="Metrics range">
                  <Select
                    value={draftMetricsPreset}
                    onValueChange={(value) =>
                      onDraftMetricsPresetChange(value as DashboardRangePreset)
                    }
                  >
                    <SelectTrigger id="team-metrics-preset">
                      <SelectValue placeholder="Range preset" />
                    </SelectTrigger>
                    <SelectContent>
                      {DASHBOARD_RANGE_PRESETS.map((preset) => (
                        <SelectItem key={preset.id} value={preset.id}>
                          {preset.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FilterField>
                <FilterField htmlFor="team-metrics-from" label="From">
                  <DatetimePicker
                    id="team-metrics-from"
                    value={draftMetricsFrom}
                    onChange={onDraftMetricsFromChange}
                  />
                </FilterField>
                <FilterField htmlFor="team-metrics-to" label="To">
                  <DatetimePicker
                    id="team-metrics-to"
                    value={draftMetricsTo}
                    onChange={onDraftMetricsToChange}
                  />
                </FilterField>
              </>
            ) : null}
            <SecondaryActionButton type="submit">Load</SecondaryActionButton>
          </DirectoryFilterForm>
        </FilterPanel>
      }
      footer={
        membersFooterVisible ? (
          <DirectoryPaginationFooter
            canGoNext={membersOffset + membersList.length < membersTotal}
            canGoPrev={membersOffset > 0}
            disabled={membersFetching}
            onNext={() => onMembersPageChange(membersOffset + membersLimit)}
            onPrev={() => onMembersPageChange(Math.max(0, membersOffset - membersLimit))}
          />
        ) : approvalsFooterVisible ? (
          <DirectoryPaginationFooter
            canGoNext={approvalsOffset + approvals.length < approvalsTotal}
            canGoPrev={approvalsOffset > 0}
            disabled={approvalsFetching}
            onNext={() => onApprovalsPageChange(approvalsOffset + approvalsLimit)}
            onPrev={() => onApprovalsPageChange(Math.max(0, approvalsOffset - approvalsLimit))}
          />
        ) : undefined
      }
      fetchState={{ fetching, error, hasSnapshot }}
      skeletonColumns={5}
      title="Team"
      badge={
        overview?.pending_approvals_count != null && overview.pending_approvals_count > 0 ? (
          <Badge variant="secondary">{overview.pending_approvals_count} pending</Badge>
        ) : undefined
      }
      alerts={<DirectoryMutationError error={actionError} />}
    >
      {overview ? (
        <div>
          <div>
            <span>{overview.customer_name ?? overview.customer_id}</span>
            {overview.cost_center ? <span>Cost center: {overview.cost_center}</span> : null}
            {overview.balance_micro != null ? (
              <span>
                Balance: {overview.balance_micro} {overview.currency ?? 'micro'}
              </span>
            ) : null}
          </div>
          {overview.license ? (
            <div>
              <span>License: {overview.license.state ?? ''}</span>
              {overview.license.plan_code ? (
                <Badge variant="outline">{overview.license.plan_code}</Badge>
              ) : null}
            </div>
          ) : null}
        </div>
      ) : null}

      {canViewTeamMetrics ? (
        <TeamMetricsPanel
          error={metricsError}
          fetching={metricsFetching}
          hasSnapshot={hasMetricsSnapshot}
          metrics={teamMetrics}
          rangeLabel={metricsRangeLabel}
        />
      ) : null}

      {showMyApprovalsPanel ? (
        <TeamMyApprovalsPanel
          error={myApprovalsError}
          fetching={myApprovalsFetching}
          hasSnapshot={hasMyApprovalsSnapshot}
          items={myApprovals}
        />
      ) : null}

      <Dialog onOpenChange={setInviteOpen} open={inviteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Invite member</DialogTitle>
          </DialogHeader>
          <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
            <FilterField htmlFor="team-invite-email" label="Email">
              <Input
                id="team-invite-email"
                type="email"
                value={draftInviteEmail}
                onChange={(event) => onDraftInviteEmailChange(event.target.value)}
              />
            </FilterField>
            <FilterField htmlFor="team-invite-role" label="Role">
              <Select onValueChange={onDraftInviteRoleChange} value={draftInviteRole}>
                <SelectTrigger className="w-full" id="team-invite-role">
                  <SelectValue placeholder="Select role" />
                </SelectTrigger>
                <SelectContent>
                  {teamRoleOptions.map((option) => (
                    <SelectItem key={option.code} value={option.code}>
                      {option.label ? `${option.code} - ${option.label}` : option.code}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FilterField>
          </DirectoryFilterForm>
          <DialogFooter>
            <PrimaryActionButton
              disabled={
                !draftCustomerId.trim() || !draftInviteEmail.trim() || !draftInviteRole.trim()
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

      <Tabs onValueChange={(value) => onRosterTabChange(value as TeamRosterTab)} value={rosterTab}>
        <TabsList>
          {ROSTER_TABS.map((item) => (
            <TabsTrigger key={item.id} value={item.id}>
              {item.label}
            </TabsTrigger>
          ))}
        </TabsList>

        <TabsContent value="members">
          <h2>Members</h2>
          {membersFetching && !hasMembersSnapshot ? (
            <p>Loading members...</p>
          ) : !membersCustomerId ? (
            <EmptyState
              title="Customer required"
              description="Load a customer to review team members."
            />
          ) : membersList.length === 0 ? (
            <EmptyState title="No members" description="Team roster is empty for this customer." />
          ) : (
            <DirectoryTable>
              <TableHeader>
                <TableRow>
                  <DirectoryTableHead>Email</DirectoryTableHead>
                  <DirectoryTableHead>Role</DirectoryTableHead>
                  <DirectoryTableHead>Campaigns</DirectoryTableHead>
                  <DirectoryTableHead>Spend cap</DirectoryTableHead>
                  <DirectoryTableHead>Blocked</DirectoryTableHead>
                  <DirectoryTableHead>Joined</DirectoryTableHead>
                  <DirectoryTableHead />
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
                        <Select
                          onValueChange={(value) => onMemberDraftChange(memberId, { role: value })}
                          value={draft.role}
                        >
                          <SelectTrigger
                            aria-label={`Role for ${member.email ?? memberId}`}
                            className="w-full"
                          >
                            <SelectValue placeholder="Role" />
                          </SelectTrigger>
                          <SelectContent>
                            {teamRoleOptions.map((option) => (
                              <SelectItem key={option.code} value={option.code}>
                                {option.label ? `${option.code} - ${option.label}` : option.code}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </TableCell>
                      <TableCell>{member.campaigns_owned ?? ''}</TableCell>
                      <TableCell>
                        <Input
                          aria-label={`Spend cap for ${member.email ?? memberId}`}
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
                      <TableCell>
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
            </DirectoryTable>
          )}

          <DirectoryFetchError
            error={membersError}
            fetchState={{
              fetching: membersFetching,
              error: membersError,
              hasSnapshot: hasMembersSnapshot,
              revalidating: membersListRevalidating,
            }}
            title="Could not load members"
          />
        </TabsContent>

        <TabsContent value="approvals">
          <h2>Budget approvals</h2>
          {approvalsFetching && !hasApprovalsSnapshot ? (
            <p>Loading approvals...</p>
          ) : !approvalsCustomerId ? (
            <EmptyState
              title="Customer required"
              description="Load a customer to review budget approvals."
            />
          ) : approvals.length === 0 ? (
            <EmptyState
              title="No pending approvals"
              description="Budget approval queue is empty."
            />
          ) : (
            <DirectoryTable>
              <TableHeader>
                <TableRow>
                  <DirectoryTableHead>Status</DirectoryTableHead>
                  <DirectoryTableHead>User</DirectoryTableHead>
                  <DirectoryTableHead>Campaign</DirectoryTableHead>
                  <DirectoryTableHead>Requested</DirectoryTableHead>
                  <DirectoryTableHead>Previous</DirectoryTableHead>
                  <DirectoryTableHead>Created</DirectoryTableHead>
                  <DirectoryTableHead />
                </TableRow>
              </TableHeader>
              <TableBody>
                {approvals.map((row) => {
                  const rowId = row.id ?? '';
                  return (
                    <TableRow key={rowId}>
                      <TableCell>{row.status ?? ''}</TableCell>
                      <TableCell>{row.user_id ?? ''}</TableCell>
                      <TableCell>{row.campaign_id ?? ''}</TableCell>
                      <TableCell>{row.requested_budget_micro ?? ''}</TableCell>
                      <TableCell>{row.previous_budget_micro ?? ''}</TableCell>
                      <TableCell>
                        {displayTimestamp(row.created_at, row.created_at_display)}
                      </TableCell>
                      <TableCell>
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
            </DirectoryTable>
          )}

          <DirectoryFetchError
            error={approvalsError}
            fetchState={{
              fetching: approvalsFetching,
              error: approvalsError,
              hasSnapshot: hasApprovalsSnapshot,
              revalidating: approvalsListRevalidating,
            }}
            title="Could not load approvals"
          />
        </TabsContent>
      </Tabs>
    </DirectoryPageShell>
  );
}
