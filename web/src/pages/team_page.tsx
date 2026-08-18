import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import type { TeamBudgetApprovalDTO, TeamMemberDTO, TeamOverviewDTO } from '../types/team.js';
import * as auth from '../helpers/auth.js';
import * as storage from '../helpers/storage.js';
import { can, isBillingReadOnly, isMediaBuyer, isTeamLead } from '../helpers/permissions.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { formatAmountMicro } from '../helpers/money.js';
import { isPageBlockingError, mapServiceError } from '../helpers/service_error.js';
import { surfaceServiceErrorToast } from '../helpers/service_error_toast.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { validateCustomerIdField } from '../helpers/validators.js';
import {
  approveTeamBudget,
  denyTeamBudget,
  fetchTeamBudgetApprovals,
  inviteTeamMember,
  updateTeamMember,
} from '../helpers/team_api.js';
import { to } from '../lib/to.js';
import { useResource } from '../helpers/use_resource.js';
import { AlertBanner } from '../components/alert_banner.js';
import { Breadcrumbs } from '../components/breadcrumbs.js';
import { BillingForecastWidget } from '../components/billing_forecast_widget.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';
import { Icon } from '../components/icon.js';
import { RecentCustomers } from '../components/recent_customers.js';
import { StatusBadge } from '../components/status_badge.js';

function teamOverviewUrl(customerId: string, sessionScoped: boolean): string | null {
  if (sessionScoped && customerId) {
    return '/api/v1/team/overview';
  }
  if (customerId) {
    return `/api/v1/team/overview?customer_id=${encodeURIComponent(customerId)}`;
  }
  return null;
}

function BlockingError({ error }: { error: unknown }) {
  if (!error) return null;
  const view = mapServiceError(error);
  if (!isPageBlockingError(view) && view.kind !== 'empty') return null;
  return <ErrorBlock error={error} />;
}

export function TeamPage() {
  const user = auth.getUser();
  const perms = user?.permissions ?? [];
  const sessionScoped = hasBoundCustomer(user?.role);
  const [searchParams] = useSearchParams();
  const [customerInput, setCustomerInput] = useState(() =>
    sessionScoped
      ? boundCustomerId(user)
      : (searchParams.get('customer_id') ?? storage.getLastCustomerId() ?? '')
  );
  const [customerInputError, setCustomerInputError] = useState<string | null>(null);
  const customerId = sessionScoped ? boundCustomerId(user) : customerInput.trim() || '';
  const teamLead = isTeamLead(user?.role);
  const mediaBuyer = isMediaBuyer(user?.role);
  const showBalance = can(perms, 'billing:read') || can(perms, 'customers:read');
  const readOnlyNote = isBillingReadOnly(perms, user?.role);
  const canManageTeam = teamLead && !mediaBuyer;

  const url = teamOverviewUrl(customerId, sessionScoped);
  const { data, loading, error, reload } = useResource<TeamOverviewDTO>(url, { skip: !url });

  useEffect(() => {
    if (error) surfaceServiceErrorToast(error);
  }, [error]);

  const members = useMemo(() => data?.members ?? [], [data?.members]);

  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState('MB');
  const [inviting, setInviting] = useState(false);
  const [approvals, setApprovals] = useState<TeamBudgetApprovalDTO[]>([]);

  const reloadApprovals = useCallback(async () => {
    if (!canManageTeam || !customerId) return;
    const [rows] = await to(fetchTeamBudgetApprovals());
    setApprovals(rows ?? []);
  }, [canManageTeam, customerId]);

  useEffect(() => {
    void reloadApprovals();
  }, [reloadApprovals]);

  const applyCustomerFilter = () => {
    const err = sessionScoped ? null : validateCustomerIdField(customerInput);
    setCustomerInputError(err);
    if (err) return;
    const id = customerInput.trim();
    if (id) storage.setLastCustomerId(id);
  };

  const submitInvite = async () => {
    if (!canManageTeam || !inviteEmail.trim()) return;
    setInviting(true);
    const [, err] = await to(inviteTeamMember(inviteEmail.trim(), inviteRole));
    setInviting(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Invite failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'Member invited', message: inviteEmail });
    setInviteEmail('');
    reload();
  };

  const setSpendCap = async (member: TeamMemberDTO, capMicro: number) => {
    const [, err] = await to(updateTeamMember(member.user_id, { spend_cap_micro: capMicro }));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Update failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'Spend cap updated', message: member.email });
    reload();
  };

  const setMemberRole = async (member: TeamMemberDTO, role: string) => {
    const [, err] = await to(updateTeamMember(member.user_id, { role }));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Role update failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'Role updated', message: member.email });
    reload();
  };

  const toggleBlocked = async (member: TeamMemberDTO, blocked: boolean) => {
    const [, err] = await to(updateTeamMember(member.user_id, { is_blocked: blocked }));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Block update failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({
      title: blocked ? 'Member blocked' : 'Member unblocked',
      message: member.email,
    });
    reload();
  };

  const resolveApproval = async (approvalId: string, approve: boolean) => {
    const fn = approve ? approveTeamBudget : denyTeamBudget;
    const [, err] = await to(fn(approvalId));
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Approval failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({
      title: approve ? 'Budget approved' : 'Budget denied',
      message: approvalId,
    });
    void reloadApprovals();
    reload();
  };

  return (
    <>
      <div className="page-header">
        <Breadcrumbs items={[{ label: 'Team' }]} />
        <div className="page-header__row">
          <div className="flex items-center gap-2">
            <Icon name="users" size={20} className="text-muted" />
            <h1 className="page-header__title">Team</h1>
          </div>
        </div>
        {readOnlyNote ? (
          <AlertBanner
            variant="info"
            message="Billing and license details are read-only for your role."
          />
        ) : null}
        {teamLead && data?.customer_id ? (
          <p className="text-muted text-sm mt-2">
            Customer:{' '}
            <Link to={`/customers/${data.customer_id}`} className="font-mono">
              {data.customer_name || data.customer_id}
            </Link>
          </p>
        ) : null}
      </div>

      {!sessionScoped ? (
        <div className="section-block mb-4">
          <RecentCustomers tenant={false} />
          <div className="filter-row mt-2">
            <input
              className={`form-input${customerInputError ? ' form-input--error' : ''}`}
              placeholder="customer_id (UUID)"
              value={customerInput}
              data-testid="team-customer-input"
              onChange={(e) => {
                setCustomerInput(e.target.value);
                setCustomerInputError(
                  e.target.value.trim() ? validateCustomerIdField(e.target.value) : null
                );
              }}
            />
            <Button label="Apply" variant="secondary" size="sm" onClick={applyCustomerFilter} />
          </div>
          {customerInputError ? <AlertBanner variant="error" message={customerInputError} /> : null}
        </div>
      ) : null}

      {!url ? (
        <AlertBanner variant="info" message="Enter a customer_id to load team overview." />
      ) : null}

      {loading ? <p className="text-muted">Loading…</p> : null}
      <BlockingError error={error} />

      {data && showBalance && (teamLead || can(perms, 'billing:read')) ? (
        <section className="section-card stack mb-6" data-testid="team-balance-panel">
          <h2 className="subsection-title">Team balance</h2>
          <div className="grid-stats">
            <div className="metric-card">
              <div className="metric-card__label">Balance</div>
              <div className="metric-card__value font-mono">
                {formatAmountMicro(data.balance_micro ?? 0, data.currency ?? 'USD')}
              </div>
            </div>
            {data.license ? (
              <div className="metric-card">
                <div className="metric-card__label">License</div>
                <div className="metric-card__value">
                  <StatusBadge status={data.license.state} />
                </div>
                {data.license.valid_until ? (
                  <p className="text-muted text-sm mt-1">
                    Valid until {new Date(data.license.valid_until).toLocaleString()}
                  </p>
                ) : null}
              </div>
            ) : null}
          </div>
          {mediaBuyer ? (
            <p className="text-muted text-sm">
              <Link to="/billing">View wallet</Link> (read-only)
            </p>
          ) : null}
        </section>
      ) : null}

      {data?.customer_id && showBalance ? (
        <div className="section-block mb-6">
          <BillingForecastWidget customerId={data.customer_id} />
        </div>
      ) : null}

      {data?.license && !showBalance ? (
        <section className="section-card stack mb-6" data-testid="team-license-panel">
          <h2 className="subsection-title">License status</h2>
          <dl className="definition-list">
            <dt>State</dt>
            <dd>
              <StatusBadge status={data.license.state} />
            </dd>
            <dt>Valid until</dt>
            <dd>
              {data.license.valid_until ? new Date(data.license.valid_until).toLocaleString() : '—'}
            </dd>
          </dl>
        </section>
      ) : null}

      {data ? (
        <section className="section-card" data-testid="team-members-panel">
          <div className="toolbar-row mb-4">
            <h2 className="subsection-title">Members</h2>
            {canManageTeam ? (
              <div className="toolbar-row" data-testid="team-invite-form">
                <input
                  className="form-input form-input--sm"
                  placeholder="email@example.com"
                  data-testid="team-invite-email"
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                />
                <select
                  className="form-input form-input--sm"
                  data-testid="team-invite-role"
                  value={inviteRole}
                  onChange={(e) => setInviteRole(e.target.value)}
                >
                  <option value="MB">Media buyer</option>
                  <option value="TL">Team lead</option>
                </select>
                <Button
                  label={inviting ? 'Inviting…' : 'Invite member'}
                  variant="primary"
                  size="sm"
                  loading={inviting}
                  data-testid="team-invite-submit"
                  onClick={() => void submitInvite()}
                />
              </div>
            ) : null}
          </div>
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">Email</th>
                  <th scope="col">Role</th>
                  <th scope="col">Campaigns owned</th>
                  <th scope="col">Spend cap</th>
                  <th scope="col">Joined</th>
                  <th scope="col">Status</th>
                  <th scope="col" />
                </tr>
              </thead>
              <tbody>
                {members.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="text-muted">
                      No team members for this customer.
                    </td>
                  </tr>
                ) : (
                  members.map((member: TeamMemberDTO) => (
                    <tr key={member.user_id} data-testid={`team-member-${member.user_id}`}>
                      <td>{member.email}</td>
                      <td>
                        {canManageTeam ? (
                          <select
                            className="form-input form-input--sm"
                            value={member.role}
                            data-testid={`team-member-role-${member.user_id}`}
                            onChange={(e) => void setMemberRole(member, e.target.value)}
                          >
                            <option value="MB">MB</option>
                            <option value="TL">TL</option>
                          </select>
                        ) : (
                          <StatusBadge status={member.role} />
                        )}
                      </td>
                      <td className="font-mono">
                        {member.campaigns_owned > 0 ? (
                          <Link to={`/campaigns?owner=${member.user_id}`}>
                            {member.campaigns_owned}
                          </Link>
                        ) : (
                          member.campaigns_owned
                        )}
                      </td>
                      <td className="font-mono">
                        {canManageTeam ? (
                          <input
                            className="form-input form-input--sm"
                            data-testid={`team-spend-cap-${member.user_id}`}
                            defaultValue={String(member.spend_cap_micro ?? 0)}
                            onBlur={(e) => {
                              const val = Number.parseInt(e.target.value, 10);
                              if (!Number.isNaN(val)) void setSpendCap(member, val);
                            }}
                          />
                        ) : (
                          formatAmountMicro(member.spend_cap_micro ?? 0, data.currency ?? 'USD')
                        )}
                      </td>
                      <td className="text-muted text-sm">
                        {member.created_at ? new Date(member.created_at).toLocaleDateString() : '—'}
                      </td>
                      <td>
                        {member.is_blocked ? (
                          <StatusBadge status="BLOCKED" kind="service" label="Blocked" />
                        ) : (
                          <StatusBadge status="ACTIVE" kind="service" label="Active" />
                        )}
                      </td>
                      <td>
                        {canManageTeam ? (
                          <Button
                            label={member.is_blocked ? 'Unblock' : 'Block'}
                            variant="secondary"
                            size="sm"
                            data-testid={`team-member-block-${member.user_id}`}
                            onClick={() => void toggleBlocked(member, !member.is_blocked)}
                          />
                        ) : null}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </section>
      ) : null}

      {canManageTeam ? (
        <section className="section-card mt-6" data-testid="team-budget-approvals">
          <h2 className="subsection-title">Budget approvals</h2>
          {approvals.length === 0 ? (
            <p className="text-muted text-sm">No pending budget approvals.</p>
          ) : (
            <div className="table-wrap">
              <table className="data-table">
                <thead>
                  <tr>
                    <th scope="col">Campaign</th>
                    <th scope="col">Requested</th>
                    <th scope="col">Previous</th>
                    <th scope="col" />
                  </tr>
                </thead>
                <tbody>
                  {approvals.map((row) => (
                    <tr key={row.id} data-testid={`team-approval-row-${row.id}`}>
                      <td className="font-mono text-sm">{row.campaign_id}</td>
                      <td>
                        {formatAmountMicro(row.requested_budget_micro, data?.currency ?? 'USD')}
                      </td>
                      <td>
                        {formatAmountMicro(row.previous_budget_micro, data?.currency ?? 'USD')}
                      </td>
                      <td className="toolbar-row">
                        <Button
                          label="Approve"
                          variant="primary"
                          size="sm"
                          data-testid={`team-approval-approve-${row.id}`}
                          onClick={() => void resolveApproval(row.id, true)}
                        />
                        <Button
                          label="Deny"
                          variant="secondary"
                          size="sm"
                          data-testid={`team-approval-deny-${row.id}`}
                          onClick={() => void resolveApproval(row.id, false)}
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      ) : null}
    </>
  );
}
