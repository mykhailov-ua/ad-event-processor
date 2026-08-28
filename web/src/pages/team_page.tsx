import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import * as auth from '../helpers/auth.js';
import { ConfirmCancelledError } from '../helpers/confirmed_api.js';
import {
  approveBudgetApproval,
  denyBudgetApproval,
  fetchTeamBudgetApprovals,
  fetchTeamOverview,
  inviteTeamMember,
  type TeamBudgetApproval,
  type TeamOverview,
} from '../helpers/team_api.js';
import { isBuyerBoundUser } from '../helpers/permissions.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { TeamHub } from '../ui/team/team_hub.js';

function parseTab(raw: string | null): string {
  if (raw === 'members' || raw === 'approvals') return raw;
  return 'overview';
}

export function TeamPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const user = auth.getUser();
  const buyerBound = isBuyerBoundUser(user?.role);
  const boundCustomerId = user?.customer_id ?? '';

  const activeTab = parseTab(searchParams.get('tab'));
  const customerId = searchParams.get('customer_id') ?? '';

  const [overview, setOverview] = useState<TeamOverview | null>(null);
  const [approvals, setApprovals] = useState<TeamBudgetApproval[]>([]);
  const [loading, setLoading] = useState(false);
  const [tabLoading, setTabLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [tabError, setTabError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [reloadToken, setReloadToken] = useState(0);

  useEffect(() => {
    if (buyerBound && boundCustomerId && !searchParams.get('customer_id')) {
      const next = new URLSearchParams(searchParams);
      next.set('customer_id', boundCustomerId);
      setSearchParams(next, { replace: true });
    }
  }, [buyerBound, boundCustomerId, searchParams, setSearchParams]);

  const reload = useCallback(() => {
    setReloadToken((token) => token + 1);
  }, []);

  useEffect(() => {
    if (!customerId) {
      setOverview(null);
      setLoading(false);
      return undefined;
    }
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [result, err] = await to(fetchTeamOverview(customerId, ctrl.signal));
      if (cancelled) return;
      if (err && err.name !== 'AbortError') setError(err);
      else setOverview(result ?? null);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [customerId, reloadToken]);

  useEffect(() => {
    if (!customerId || activeTab !== 'approvals') {
      setApprovals([]);
      return undefined;
    }
    const ctrl = new AbortController();
    let cancelled = false;
    setTabLoading(true);
    setTabError(null);
    void (async () => {
      const [result, err] = await to(fetchTeamBudgetApprovals(customerId, ctrl.signal));
      if (cancelled) return;
      if (err && err.name !== 'AbortError') setTabError(err);
      else setApprovals(result ?? []);
      setTabLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [customerId, activeTab, reloadToken]);

  const onTabChange = useCallback(
    (tabId: string) => {
      const next = new URLSearchParams(searchParams);
      if (tabId === 'overview') next.delete('tab');
      else next.set('tab', tabId);
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  const onCustomerApply = useCallback(
    (nextCustomerId: string) => {
      const next = new URLSearchParams(searchParams);
      if (nextCustomerId) next.set('customer_id', nextCustomerId);
      else next.delete('customer_id');
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  const onInvite = useCallback(
    async (body: { email: string; role: string }) => {
      if (!customerId) return;
      setBusy(true);
      try {
        await inviteTeamMember({ ...body, customer_id: customerId });
        pushToastMessage({ title: 'Member invited', message: body.email });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Invite failed',
          message: err instanceof Error ? err.message : 'Invite failed',
        });
      } finally {
        setBusy(false);
      }
    },
    [customerId, reload]
  );

  const onApprove = useCallback(
    async (id: string) => {
      if (!customerId) return;
      setBusy(true);
      try {
        await approveBudgetApproval(id, customerId);
        pushToastMessage({ title: 'Approval granted', message: id });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Approve failed',
          message: err instanceof Error ? err.message : 'Approve failed',
        });
      } finally {
        setBusy(false);
      }
    },
    [customerId, reload]
  );

  const onDeny = useCallback(
    async (id: string) => {
      if (!customerId) return;
      setBusy(true);
      try {
        await denyBudgetApproval(id, customerId);
        pushToastMessage({ title: 'Approval denied', message: id });
        reload();
      } catch (err) {
        if (err instanceof ConfirmCancelledError) return;
        pushToastMessage({
          title: 'Deny failed',
          message: err instanceof Error ? err.message : 'Deny failed',
        });
      } finally {
        setBusy(false);
      }
    },
    [customerId, reload]
  );

  return (
    <TeamHub
      activeTab={activeTab}
      customerId={customerId}
      overview={overview}
      approvals={approvals}
      loading={loading}
      tabLoading={tabLoading}
      error={error}
      tabError={tabError}
      busy={busy}
      onTabChange={onTabChange}
      onCustomerApply={onCustomerApply}
      onInvite={(body) => {
        void onInvite(body);
      }}
      onApprove={(id) => {
        void onApprove(id);
      }}
      onDeny={(id) => {
        void onDeny(id);
      }}
    />
  );
}
