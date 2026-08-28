import { useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import type {
  TeamBudgetApproval,
  TeamMember,
  TeamOverview,
} from '../../helpers/team_api.js';
import * as auth from '../../helpers/auth.js';
import { can, canReadCampaigns } from '../../helpers/permissions.js';
import { formatAmountMicro } from '../../helpers/money.js';
import { CustomerScopeBar } from '../integrations/customer_scope_bar.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { TabBar } from '../system/tab_bar.js';
import styles from '../settings/settings_shared.module.css';

const TEAM_TABS = [
  { id: 'overview', label: 'Overview' },
  { id: 'members', label: 'Members' },
  { id: 'approvals', label: 'Approvals' },
];

export type TeamHubProps = {
  activeTab: string;
  customerId: string;
  overview: TeamOverview | null;
  approvals: TeamBudgetApproval[];
  loading: boolean;
  tabLoading: boolean;
  error: unknown;
  tabError: unknown;
  busy: boolean;
  onTabChange: (tabId: string) => void;
  onCustomerApply: (customerId: string) => void;
  onInvite: (body: { email: string; role: string }) => void;
  onApprove: (id: string) => void;
  onDeny: (id: string) => void;
};

export function TeamHub({
  activeTab,
  customerId,
  overview,
  approvals,
  loading,
  tabLoading,
  error,
  tabError,
  busy,
  onTabChange,
  onCustomerApply,
  onInvite,
  onApprove,
  onDeny,
}: TeamHubProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const canView =
    canReadCampaigns(permissions) || can(permissions, 'billing:read');
  const canWrite = can(permissions, 'campaigns:write') || can(permissions, 'users:write');

  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState('MB');

  if (!canView) {
    return <ErrorBlock error={new Error('forbidden')} fallbackTitle="Team access denied" />;
  }

  if (error && !overview && activeTab === 'overview') {
    return <ErrorBlock error={error} fallbackTitle="Failed to load team overview" />;
  }

  const members = overview?.members ?? [];

  const onInviteSubmit = (event: FormEvent) => {
    event.preventDefault();
    const email = inviteEmail.trim();
    if (!email) return;
    onInvite({ email, role: inviteRole });
    setInviteEmail('');
  };

  return (
    <div className={styles.root} data-testid="team-page">
      <PageChrome
        title="Team"
        badge={
          <Link to="/settings" className={styles.bannerLink}>
            Settings
          </Link>
        }
      />
      <CustomerScopeBar customerId={customerId} onApply={onCustomerApply} />
      <div className={styles.toolbar}>
        <TabBar tabs={TEAM_TABS} active={activeTab} onChange={onTabChange} />
      </div>

      {!customerId ? (
        <p className={styles.hint}>Enter a customer ID and apply to load team data.</p>
      ) : null}

      {activeTab === 'overview' && customerId ? (
        loading && !overview ? (
          <PageSkeleton rows={3} />
        ) : overview ? (
          <div className={styles.content}>
            <div className={styles.kpiRow}>
              <div className={styles.kpiTile}>
                <p className={styles.kpiLabel}>Customer</p>
                <p className={styles.kpiValue}>{overview.customer_name ?? overview.customer_id}</p>
              </div>
              <div className={styles.kpiTile}>
                <p className={styles.kpiLabel}>Members</p>
                <p className={styles.kpiValue}>{String(members.length)}</p>
              </div>
              {overview.balance_micro != null ? (
                <div className={styles.kpiTile}>
                  <p className={styles.kpiLabel}>Balance</p>
                  <p className={styles.kpiValue}>
                    {formatAmountMicro(overview.balance_micro, overview.currency ?? 'USD')}
                  </p>
                </div>
              ) : null}
              {overview.license ? (
                <div className={styles.kpiTile}>
                  <p className={styles.kpiLabel}>License</p>
                  <p className={styles.kpiValue}>{overview.license.state ?? '-'}</p>
                </div>
              ) : null}
            </div>
            {overview.cost_center ? (
              <p className={styles.hint}>Cost center: {overview.cost_center}</p>
            ) : null}
          </div>
        ) : null
      ) : null}

      {activeTab === 'members' && customerId ? (
        <div className={styles.content}>
          {loading && members.length === 0 ? (
            <PageSkeleton rows={4} columns={6} />
          ) : members.length === 0 ? (
            <EmptyState message="No team members returned." />
          ) : (
            <div className={`${styles.gridTable} ${styles.colsMembers}`} role="grid">
              <div className={styles.gridHeader} role="row">
                <span className={styles.gridCell} role="columnheader">
                  Email
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Role
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Campaigns
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Spend cap
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Blocked
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Joined
                </span>
              </div>
              {members.map((row: TeamMember) => (
                <div key={row.user_id ?? row.email} className={styles.gridRow} role="row">
                  <span className={styles.gridCell} role="gridcell">
                    {row.email ?? '-'}
                  </span>
                  <span className={styles.gridCell} role="gridcell">
                    {row.role ?? '-'}
                  </span>
                  <span className={styles.gridCell} role="gridcell">
                    {row.campaigns_owned ?? '-'}
                  </span>
                  <span className={styles.gridCell} role="gridcell">
                    {row.spend_cap_micro != null
                      ? formatAmountMicro(row.spend_cap_micro, overview?.currency ?? 'USD')
                      : '-'}
                  </span>
                  <span className={styles.gridCell} role="gridcell">
                    {row.is_blocked ? 'yes' : 'no'}
                  </span>
                  <span className={styles.gridCell} role="gridcell">
                    {row.created_at ?? '-'}
                  </span>
                </div>
              ))}
            </div>
          )}
          {canWrite ? (
            <form className={styles.formStack} onSubmit={onInviteSubmit}>
              <h3 className={styles.sectionTitle}>Invite member</h3>
              <label className={styles.field}>
                <span className={styles.fieldLabel}>Email</span>
                <input
                  className={styles.textInput}
                  type="email"
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                />
              </label>
              <label className={styles.field}>
                <span className={styles.fieldLabel}>Role</span>
                <select
                  className={styles.select}
                  value={inviteRole}
                  onChange={(e) => setInviteRole(e.target.value)}
                >
                  <option value="MB">Media buyer</option>
                  <option value="U">User</option>
                  <option value="A">Admin</option>
                </select>
              </label>
              <Button type="submit" variant="primary" disabled={busy || !inviteEmail.trim()}>
                Invite
              </Button>
            </form>
          ) : null}
        </div>
      ) : null}

      {activeTab === 'approvals' && customerId ? (
        <div className={styles.content}>
          {tabError ? (
            <ErrorBlock error={tabError} fallbackTitle="Failed to load budget approvals" />
          ) : tabLoading && approvals.length === 0 ? (
            <PageSkeleton rows={4} columns={6} />
          ) : approvals.length === 0 ? (
            <EmptyState message="No pending budget approvals." />
          ) : (
            <div className={`${styles.gridTable} ${styles.colsApprovals}`} role="grid">
              <div className={styles.gridHeader} role="row">
                <span className={styles.gridCell} role="columnheader">
                  User
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Campaign
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Requested
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Previous
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Status
                </span>
                <span className={styles.gridCell} role="columnheader">
                  Actions
                </span>
              </div>
              {approvals.map((row) => (
                <div key={row.id} className={styles.gridRow} role="row">
                  <span className={styles.gridCell} role="gridcell">
                    {row.user_id ?? '-'}
                  </span>
                  <span className={styles.gridCell} role="gridcell">
                    {row.campaign_id ?? '-'}
                  </span>
                  <span className={styles.gridCell} role="gridcell">
                    {formatAmountMicro(row.requested_budget_micro, overview?.currency ?? 'USD')}
                  </span>
                  <span className={styles.gridCell} role="gridcell">
                    {formatAmountMicro(row.previous_budget_micro, overview?.currency ?? 'USD')}
                  </span>
                  <span className={styles.gridCell} role="gridcell">
                    {row.status ?? '-'}
                  </span>
                  <span className={`${styles.gridCell} ${styles.actions}`} role="gridcell">
                    {canWrite && row.id && row.status === 'pending' ? (
                      <>
                        <Button
                          type="button"
                          size="sm"
                          disabled={busy}
                          onClick={() => onApprove(row.id!)}
                        >
                          Approve
                        </Button>
                        <Button
                          type="button"
                          size="sm"
                          variant="danger"
                          disabled={busy}
                          onClick={() => onDeny(row.id!)}
                        >
                          Deny
                        </Button>
                      </>
                    ) : (
                      '-'
                    )}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      ) : null}
    </div>
  );
}
