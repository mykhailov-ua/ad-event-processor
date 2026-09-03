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
        <div className="flex flex-col gap-3 flex flex-col gap-2">
          <div aria-label="Ops sections" className="flex flex-wrap items-center gap-2 flex flex-wrap items-center gap-2 border-t border-zinc-200 pt-2 dark:border-zinc-800">
            <OpsNav variant="admin" />
          </div>
          {actions ? (
            <div aria-label="Ops actions" className="flex flex-wrap items-center gap-2" role="toolbar">
              {actions}
            </div>
          ) : null}
          {filters ? (
            <div aria-label="Ops filters" className="flex flex-wrap items-center gap-2 flex flex-wrap items-center gap-2" role="search">
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
    <div aria-label={label} className="flex flex-wrap items-center gap-1">
      {children}
    </div>
  );
}
