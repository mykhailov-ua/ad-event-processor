import type { ReactNode } from 'react';

import { adminSpacing, opsControlPanelClass } from '@/lib/admin_spacing';
import { PageLayout } from '@/shell/page_layout';
import { PageSkeleton } from '@/shell/page_skeleton';
import { panelError } from '@/shell/panel_error';
import {
  type DirectoryFetchState,
  resolveDirectoryLoadPhase,
  shouldShowDirectoryRefreshError,
} from '@/shell/directory_load_state';
import { OpsNav } from '@/domains/ops/ops_nav';

export type OpsPageShellProps = {
  title: string;
  badge?: ReactNode;
  /** Primary actions row (reload, export, load, etc.). Wrap groups in flex with gap-1. */
  actions?: ReactNode;
  /** Filter controls row (Label / Input from components/ui). */
  filters?: ReactNode;
  footer?: ReactNode;
  children: ReactNode;
};

export function OpsPageShell({
  title,
  badge,
  actions,
  filters,
  footer,
  children,
}: OpsPageShellProps) {
  return (
    <PageLayout
      badge={badge}
      controlPanel={
        <div className={opsControlPanelClass}>
          <div aria-label="Ops sections" className={adminSpacing.flex.buttonGroup}>
            <OpsNav variant="admin" />
          </div>
          {actions ? (
            <div
              aria-label="Ops actions"
              className={adminSpacing.flex.buttonGroup}
              role="toolbar"
            >
              {actions}
            </div>
          ) : null}
          {filters ? (
            <div
              aria-label="Ops filters"
              className={adminSpacing.grid.filterMatrix}
              role="search"
            >
              {filters}
            </div>
          ) : null}
        </div>
      }
      footer={footer}
      title={title}
    >
      {children}
    </PageLayout>
  );
}

export function OpsPageLoading({ columns = 4 }: { columns?: number } = {}) {
  return <PageSkeleton columns={columns} variant="directory" />;
}

export function OpsPageBlockingError({
  pageTitle,
  title,
  error,
}: {
  pageTitle: string;
  title: string;
  error: Error;
}) {
  return (
    <OpsPageShell title={pageTitle}>
      {panelError(error, title)}
    </OpsPageShell>
  );
}

export function OpsPageRefreshError({
  error,
  fetchState,
  title = 'Refresh failed',
}: {
  error: Error | null | undefined;
  fetchState: DirectoryFetchState;
  title?: string;
}) {
  if (!error || !shouldShowDirectoryRefreshError(fetchState)) {
    return null;
  }
  return panelError(error, title);
}

export function resolveOpsPagePhase(fetchState: DirectoryFetchState) {
  return resolveDirectoryLoadPhase(fetchState);
}

export type OpsPageWithLoadProps = {
  title: string;
  badge?: ReactNode;
  actions?: ReactNode;
  filters?: ReactNode;
  footer?: ReactNode;
  blockingErrorTitle: string;
  refreshErrorTitle?: string;
  skeletonColumns?: number;
  fetchState: DirectoryFetchState;
  /** Non-fetch errors (mutation, export) and status lines above main content. */
  alerts?: ReactNode;
  children: ReactNode;
};

export function OpsPageWithLoad({
  title,
  badge,
  actions,
  filters,
  footer,
  blockingErrorTitle,
  refreshErrorTitle,
  skeletonColumns = 4,
  fetchState,
  alerts,
  children,
}: OpsPageWithLoadProps) {
  const phase = resolveDirectoryLoadPhase(fetchState);

  if (phase === 'loading') {
    return <OpsPageLoading columns={skeletonColumns} />;
  }

  if (phase === 'blocking-error' && fetchState.error) {
    return (
      <OpsPageBlockingError
        error={fetchState.error}
        pageTitle={title}
        title={blockingErrorTitle}
      />
    );
  }

  return (
    <OpsPageShell
      badge={badge}
      actions={actions}
      filters={filters}
      footer={footer}
      title={title}
    >
      <OpsPageRefreshError
        error={fetchState.error}
        fetchState={fetchState}
        title={refreshErrorTitle}
      />
      {alerts}
      {children}
    </OpsPageShell>
  );
}

/** Group action buttons the same way as campaigns_list_toolbar. */
export function OpsActionGroup({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div aria-label={label} className={adminSpacing.flex.buttonGroup}>
      {children}
    </div>
  );
}
