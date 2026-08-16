import { useEffect, useMemo } from 'react';
import { Link } from 'react-router-dom';
import type { TeamMemberDTO, TeamOverviewDTO } from '../types/api/team.js';
import * as auth from '../helpers/auth.js';
import { can, isBillingReadOnly, isMediaBuyer, isTeamLead } from '../helpers/permissions.js';
import { hasBoundCustomer, boundCustomerId } from '../helpers/buyer_session.js';
import { formatAmountMicro } from '../helpers/money.js';
import { isPageBlockingError, mapServiceError } from '../helpers/service_error.js';
import { surfaceServiceErrorToast } from '../helpers/service_error_toast.js';
import { useResource } from '../hooks/use_resource.js';
import { AlertBanner } from '../components/alert_banner.js';
import { Breadcrumbs } from '../components/breadcrumbs.js';
import { ErrorBlock } from '../components/error_block.js';
import { Icon } from '../components/icon.js';
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

/**
 * Team members, license status, and team balance (RBAC-gated).
 */
export function TeamPage() {
  const user = auth.getUser();
  const perms = user?.permissions ?? [];
  const sessionScoped = hasBoundCustomer(user?.role);
  const customerId = sessionScoped ? boundCustomerId(user) : '';
  const teamLead = isTeamLead(user?.role);
  const mediaBuyer = isMediaBuyer(user?.role);
  const showBalance = can(perms, 'billing:read') || can(perms, 'customers:read');
  const readOnlyNote = isBillingReadOnly(perms, user?.role);

  const url = teamOverviewUrl(customerId, sessionScoped);
  const { data, loading, error } = useResource<TeamOverviewDTO>(url, { skip: !url });

  useEffect(() => {
    if (error) surfaceServiceErrorToast(error);
  }, [error]);

  const members = useMemo(() => data?.members ?? [], [data?.members]);

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

      {!url ? (
        <AlertBanner variant="info" message="Sign in with a team-scoped role or select a customer." />
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
              <Link to="/billing">View wallet</Link>
              {' '}
              (read-only)
            </p>
          ) : null}
        </section>
      ) : null}

      {data?.license && !showBalance ? (
        <section className="section-card stack mb-6" data-testid="team-license-panel">
          <h2 className="subsection-title">License status</h2>
          <dl className="definition-list">
            <dt>State</dt>
            <dd><StatusBadge status={data.license.state} /></dd>
            <dt>Valid until</dt>
            <dd>{data.license.valid_until ? new Date(data.license.valid_until).toLocaleString() : '—'}</dd>
          </dl>
        </section>
      ) : null}

      {data ? (
        <section className="section-card" data-testid="team-members-panel">
          <h2 className="subsection-title">Members</h2>
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">Email</th>
                  <th scope="col">Role</th>
                  <th scope="col">Campaigns owned</th>
                  <th scope="col">Joined</th>
                </tr>
              </thead>
              <tbody>
                {members.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="text-muted">No team members for this customer.</td>
                  </tr>
                ) : (
                  members.map((member: TeamMemberDTO) => (
                    <tr key={member.user_id}>
                      <td>{member.email}</td>
                      <td><StatusBadge status={member.role} /></td>
                      <td className="font-mono">{member.campaigns_owned}</td>
                      <td className="text-muted text-sm">
                        {member.created_at ? new Date(member.created_at).toLocaleDateString() : '—'}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </section>
      ) : null}
    </>
  );
}
