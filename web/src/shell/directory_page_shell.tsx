import type { ReactNode } from 'react';

import { adminSpacing } from '@/lib/admin_spacing';
import { cn } from '@/lib/utils';
import { PageLayout } from '@/shell/page_layout';
import { PageSkeleton } from '@/shell/page_skeleton';
import { AdminMutationError } from '@/shell/admin_error';
import { panelError } from '@/shell/panel_error';
import {
  type DirectoryFetchState,
  resolveDirectoryLoadPhase,
  shouldShowDirectoryRefreshError,
} from '@/shell/directory_load_state';

export type DirectoryPageShellProps = {
  title: ReactNode;
  description?: ReactNode;
  badge?: ReactNode;
  actions?: ReactNode;
  controlPanel?: ReactNode;
  footer?: ReactNode;
  blockingErrorTitle: string;
  blockingErrorFooter?: ReactNode;
  refreshErrorTitle?: string;
  refreshErrorFooter?: ReactNode;
  skeletonColumns?: number;
  fetchState: DirectoryFetchState;
  /** Non-fetch errors (export, mutation) and status lines above main content. */
  alerts?: ReactNode;
  headerClassName?: string;
  workspaceClassName?: string;
  mainClassName?: string;
  footerClassName?: string;
  fillViewport?: boolean;
  aside?: ReactNode;
  asideClassName?: string;
  children: ReactNode;
};

export function DirectoryPageShell({
  title,
  description,
  badge,
  actions,
  controlPanel,
  footer,
  blockingErrorTitle,
  blockingErrorFooter,
  refreshErrorTitle = 'Refresh failed',
  refreshErrorFooter,
  skeletonColumns = 4,
  fetchState,
  alerts,
  headerClassName,
  workspaceClassName,
  mainClassName,
  footerClassName,
  fillViewport,
  aside,
  asideClassName,
  children,
}: DirectoryPageShellProps) {
  const phase = resolveDirectoryLoadPhase(fetchState);

  if (phase === 'loading') {
    return <PageSkeleton columns={skeletonColumns} variant="directory" />;
  }

  if (phase === 'blocking-error' && fetchState.error) {
    return (
      <div className={cn('grid', adminSpacing.gap.md)}>
        {panelError(fetchState.error, blockingErrorTitle)}
        {blockingErrorFooter}
      </div>
    );
  }

  return (
    <PageLayout
      aside={aside}
      asideClassName={asideClassName}
      badge={badge}
      controlPanel={controlPanel}
      description={description}
      fillViewport={fillViewport}
      footer={footer}
      footerClassName={footerClassName}
      headerActions={actions}
      headerClassName={headerClassName}
      mainClassName={mainClassName}
      title={title}
      workspaceClassName={workspaceClassName}
    >
      {shouldShowDirectoryRefreshError(fetchState) && fetchState.error ? (
        <div className={cn('grid', adminSpacing.gap.md)}>
          {panelError(fetchState.error, refreshErrorTitle)}
          {refreshErrorFooter}
        </div>
      ) : null}
      {alerts}
      {children}
    </PageLayout>
  );
}

export function DirectoryMutationError({
  error,
  title = 'Action failed',
}: {
  error: Error | null | undefined;
  title?: string;
}) {
  if (!error) {
    return null;
  }
  return <AdminMutationError error={error} title={title} />;
}

/** Inline fetch error for nested panels (tabs, secondary lists) without replacing the page shell. */
export function DirectoryFetchError({
  error,
  title,
  refreshTitle = 'Refresh failed',
  fetchState,
}: {
  error: Error | null | undefined;
  title: string;
  refreshTitle?: string;
  fetchState: DirectoryFetchState;
}) {
  const phase = resolveDirectoryLoadPhase(fetchState);
  if (!error || phase === 'loading') {
    return null;
  }
  if (phase === 'blocking-error') {
    return panelError(error, title);
  }
  if (shouldShowDirectoryRefreshError(fetchState)) {
    return panelError(error, refreshTitle);
  }
  return null;
}
