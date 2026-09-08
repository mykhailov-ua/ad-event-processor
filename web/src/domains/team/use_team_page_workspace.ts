// L3 team admin: roster tab + budget approvals; separate refresh tokens per lane.
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';

import {
  approveTeamBudgetApproval,
  denyTeamBudgetApproval,
  getTeamOverview,
  inviteTeamMember,
  listTeamBudgetApprovals,
  listTeamMembers,
  updateTeamMember,
} from '@/api/team_api';
import { refreshSession } from '@/api/auth_api';
import type { TeamMemberEditDraft, TeamRosterTab } from '@/domains/team/team_overview';
import { confirmDestructiveAction, mutationError } from '@/lib/mutation_audit';
import { useResource } from '@/api/use_resource';
import { useSession } from '@/hooks/use_session';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { parseListLimit, parseListOffset } from '@/lib/list_query';

export function useTeamPageWorkspace() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const { session, user, refetchSession } = useSession();
  const [rosterTab, setRosterTab] = useState<TeamRosterTab>('members');
  const [overviewRefreshToken, setOverviewRefreshToken] = useState(0);
  const [rosterRefreshToken, setRosterRefreshToken] = useState(0);
  const [actingId, setActingId] = useState<string | undefined>();
  const [memberUpdatingId, setMemberUpdatingId] = useState<string | undefined>();
  const [actionError, setActionError] = useState<Error | undefined>();
  const [inviting, setInviting] = useState(false);
  const [inviteSuccess, setInviteSuccess] = useState(false);
  const [memberDrafts, setMemberDrafts] = useState<Record<string, TeamMemberEditDraft>>({});

  const appliedCustomerId = searchParams.get('customer_id') ?? session?.default_customer_id ?? '';
  const appliedMembersLimit = parseListLimit(searchParams.get('member_limit'), 100);
  const appliedMembersOffset = parseListOffset(searchParams.get('member_offset'));
  const appliedApprovalsLimit = parseListLimit(searchParams.get('approval_limit'), 100);
  const appliedApprovalsOffset = parseListOffset(searchParams.get('approval_offset'));

  const [draftCustomerId, setDraftCustomerId] = useState(appliedCustomerId);
  const [draftInviteEmail, setDraftInviteEmail] = useState('');
  const [draftInviteRole, setDraftInviteRole] = useState('MB');

  useEffect(() => {
    setDraftCustomerId(appliedCustomerId);
  }, [appliedCustomerId]);

  const { data, error, fetching } = useResource(
    (signal) => getTeamOverview({ customer_id: appliedCustomerId || undefined }, signal),
    [appliedCustomerId, overviewRefreshToken]
  );

  const shouldFetchMembers = Boolean(appliedCustomerId) && rosterTab === 'members';

  const {
    data: membersData,
    error: membersError,
    fetching: membersFetching,
    revalidating: membersRevalidating,
  } = useResource(
    (signal) => {
      if (!shouldFetchMembers) {
        return Promise.resolve(undefined);
      }
      return listTeamMembers(
        {
          customer_id: appliedCustomerId,
          limit: appliedMembersLimit,
          offset: appliedMembersOffset,
        },
        signal
      );
    },
    [
      appliedCustomerId,
      appliedMembersLimit,
      appliedMembersOffset,
      rosterRefreshToken,
      shouldFetchMembers,
      rosterTab,
    ]
  );

  useEffect(() => {
    if (!membersData?.items) {
      return;
    }
    setMemberDrafts((prev) => {
      const next = { ...prev };
      for (const member of membersData.items ?? []) {
        const memberId = member.user_id ?? '';
        if (!memberId || next[memberId]) {
          continue;
        }
        next[memberId] = {
          role: member.role ?? '',
          is_blocked: member.is_blocked ?? false,
          spend_cap_micro: member.spend_cap_micro != null ? String(member.spend_cap_micro) : '',
        };
      }
      return next;
    });
  }, [membersData?.items]);

  const shouldFetchApprovals = Boolean(appliedCustomerId) && rosterTab === 'approvals';

  const {
    data: approvalsData,
    error: approvalsError,
    fetching: approvalsFetching,
    revalidating: approvalsRevalidating,
  } = useResource(
    (signal) => {
      if (!shouldFetchApprovals) {
        return Promise.resolve(undefined);
      }
      return listTeamBudgetApprovals(
        {
          customer_id: appliedCustomerId,
          limit: appliedApprovalsLimit,
          offset: appliedApprovalsOffset,
        },
        signal
      );
    },
    [
      appliedApprovalsLimit,
      appliedApprovalsOffset,
      appliedCustomerId,
      rosterRefreshToken,
      shouldFetchApprovals,
      rosterTab,
    ]
  );

  const updateTeamQuery = useCallback(
    (patch: {
      customer_id?: string;
      member_limit?: number;
      member_offset?: number;
      approval_limit?: number;
      approval_offset?: number;
    }) => {
      const next = new URLSearchParams(searchParams);
      const customerId = patch.customer_id ?? appliedCustomerId;
      const memberLimit = patch.member_limit ?? appliedMembersLimit;
      const memberOffset = patch.member_offset ?? appliedMembersOffset;
      const approvalLimit = patch.approval_limit ?? appliedApprovalsLimit;
      const approvalOffset = patch.approval_offset ?? appliedApprovalsOffset;

      if (customerId) {
        next.set('customer_id', customerId);
      } else {
        next.delete('customer_id');
      }
      next.set('member_limit', String(memberLimit));
      next.set('member_offset', String(Math.max(0, memberOffset)));
      next.set('approval_limit', String(approvalLimit));
      next.set('approval_offset', String(Math.max(0, approvalOffset)));
      replaceSearchParams(next);
    },
    [
      appliedApprovalsLimit,
      appliedApprovalsOffset,
      appliedCustomerId,
      appliedMembersLimit,
      appliedMembersOffset,
      replaceSearchParams,
      searchParams,
    ]
  );

  const onApplyCustomer = useCallback(() => {
    updateTeamQuery({
      customer_id: draftCustomerId.trim(),
      member_offset: 0,
      approval_offset: 0,
    });
  }, [draftCustomerId, updateTeamQuery]);

  const onMembersPageChange = useCallback(
    (nextOffset: number) => {
      updateTeamQuery({ member_offset: Math.max(0, nextOffset) });
    },
    [updateTeamQuery]
  );

  const onApprovalsPageChange = useCallback(
    (nextOffset: number) => {
      updateTeamQuery({ approval_offset: Math.max(0, nextOffset) });
    },
    [updateTeamQuery]
  );

  const onMemberDraftChange = useCallback(
    (memberId: string, patch: Partial<TeamMemberEditDraft>) => {
      setMemberDrafts((prev) => ({
        ...prev,
        [memberId]: {
          role: prev[memberId]?.role ?? '',
          is_blocked: prev[memberId]?.is_blocked ?? false,
          spend_cap_micro: prev[memberId]?.spend_cap_micro ?? '',
          ...patch,
        },
      }));
    },
    []
  );

  const onSaveMember = useCallback(
    async (memberId: string) => {
      const customerId = appliedCustomerId.trim();
      const draft = memberDrafts[memberId];
      if (!customerId || !memberId || !draft) {
        return;
      }
      const spendRaw = draft.spend_cap_micro.trim();
      const spendCapMicro = spendRaw ? Number.parseInt(spendRaw, 10) : undefined;
      if (spendRaw && (!Number.isFinite(spendCapMicro) || spendCapMicro! < 0)) {
        setActionError(new Error('spend_cap_micro must be a non-negative integer'));
        return;
      }

      setMemberUpdatingId(memberId);
      setActionError(undefined);
      try {
        await updateTeamMember(customerId, memberId, {
          role: draft.role.trim() || undefined,
          is_blocked: draft.is_blocked,
          spend_cap_micro: spendCapMicro,
        });
        if (user?.id === memberId) {
          await refreshSession();
          refetchSession();
        }
        toast.success('Member updated');
        setRosterRefreshToken((value) => value + 1);
      } catch (err: unknown) {
        const nextError = mutationError(err);
        setActionError(nextError);
        toast.error(nextError.message);
      } finally {
        setMemberUpdatingId(undefined);
      }
    },
    [appliedCustomerId, memberDrafts, refetchSession, user?.id]
  );

  const onInvite = useCallback(async () => {
    if (inviting) {
      return;
    }
    const customerId = appliedCustomerId.trim();
    const email = draftInviteEmail.trim();
    const role = draftInviteRole.trim();
    if (!customerId || !email || !role) {
      return;
    }
    setInviting(true);
    setActionError(undefined);
    setInviteSuccess(false);
    try {
      await inviteTeamMember(customerId, { email, role });
      setInviteSuccess(true);
      setDraftInviteEmail('');
      toast.success('Invite sent');
      setRosterRefreshToken((value) => value + 1);
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setActionError(nextError);
      toast.error(nextError.message);
    } finally {
      setInviting(false);
    }
  }, [appliedCustomerId, draftInviteEmail, draftInviteRole, inviting]);

  const runApprovalAction = useCallback(async (id: string, action: 'approve' | 'deny') => {
    if (action === 'deny' && !confirmDestructiveAction('Deny this budget approval request?')) {
      return;
    }
    setActingId(id);
    setActionError(undefined);
    try {
      if (action === 'approve') {
        await approveTeamBudgetApproval(id);
        toast.success('Budget approval granted');
      } else {
        await denyTeamBudgetApproval(id);
        toast.success('Budget approval denied');
      }
      setRosterRefreshToken((value) => value + 1);
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setActionError(nextError);
      toast.error(nextError.message);
    } finally {
      setActingId(undefined);
    }
  }, []);

  return {
    rosterTab,
    onRosterTabChange: setRosterTab,
    overview: data,
    members: membersData?.items ?? [],
    membersTotal: membersData?.total ?? 0,
    membersLimit: membersData?.limit ?? appliedMembersLimit,
    membersOffset: membersData?.offset ?? appliedMembersOffset,
    membersCustomerId: appliedCustomerId,
    approvals: approvalsData?.items ?? [],
    approvalsTotal: approvalsData?.total ?? 0,
    approvalsLimit: approvalsData?.limit ?? appliedApprovalsLimit,
    approvalsOffset: approvalsData?.offset ?? appliedApprovalsOffset,
    approvalsCustomerId: appliedCustomerId,
    draftCustomerId,
    draftInviteEmail,
    draftInviteRole,
    memberDrafts,
    fetching,
    membersFetching,
    membersListRevalidating: membersRevalidating || listQueryPending,
    approvalsFetching,
    approvalsListRevalidating: approvalsRevalidating || listQueryPending,
    inviting,
    error,
    membersError,
    approvalsError,
    actionError,
    inviteSuccess,
    hasSnapshot: data != null,
    hasMembersSnapshot: !shouldFetchMembers || membersData != null,
    hasApprovalsSnapshot: !shouldFetchApprovals || approvalsData != null,
    actingId,
    memberUpdatingId,
    onDraftCustomerIdChange: setDraftCustomerId,
    onDraftInviteEmailChange: setDraftInviteEmail,
    onDraftInviteRoleChange: setDraftInviteRole,
    onMemberDraftChange,
    onApplyCustomer,
    onInvite,
    onMembersPageChange,
    onApprovalsPageChange,
    onSaveMember,
    onApprove: (id: string) => void runApprovalAction(id, 'approve'),
    onDeny: (id: string) => void runApprovalAction(id, 'deny'),
  };
}
