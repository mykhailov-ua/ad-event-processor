import type { ReactNode } from 'react';

import { ErrorBlock } from '@/shell/error_block';
import { PageLayout } from '@/shell/page_layout';
import { PageSkeleton } from '@/shell/page_skeleton';
import { OpsNav } from '@/domains/ops/ops_nav';

export type OpsPageShellProps = {
  title: string;
  badge?: ReactNode;
  /** Primary actions row (reload, export, load, etc.). Wrap groups in admin-toolbar-group. */
  actions?: ReactNode;
  /** Filter controls row (admin-label / admin-input). */
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
        <div className="admin-stack admin-stack--compact">
          <div aria-label="Ops sections" className="admin-toolbar-row admin-toolbar-row--sections">
            <OpsNav variant="admin" />
          </div>
          {actions ? (
            <div aria-label="Ops actions" className="admin-toolbar-row" role="toolbar">
              {actions}
            </div>
          ) : null}
          {filters ? (
            <div aria-label="Ops filters" className="admin-toolbar-row admin-toolbar-row--filters" role="search">
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

export function OpsPageLoading() {
  return <PageSkeleton />;
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
      <ErrorBlock error={error} title={title} />
    </OpsPageShell>
  );
}

/** Group action buttons the same way as campaigns_list_toolbar. */
export function OpsActionGroup({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  return (
    <div aria-label={label} className="admin-toolbar-group">
      {children}
    </div>
  );
}
