import type { ReactNode } from 'react';

import { PageSkeleton } from '@/shell/page_skeleton';
import { panelError, type PanelErrorOptions } from '@/shell/panel_error';
import {
  type DirectoryFetchState,
  resolveDirectoryLoadPhase,
  shouldShowDirectoryRefreshError,
} from '@/shell/directory_load_state';

export type CustomerTabShellProps = {
  title?: ReactNode;
  fetchState: DirectoryFetchState;
  blockingErrorTitle: string;
  refreshErrorTitle?: string;
  blockingErrorOptions?: PanelErrorOptions;
  children: ReactNode;
};

export function CustomerTabShell({
  title,
  fetchState,
  blockingErrorTitle,
  refreshErrorTitle = 'Refresh failed',
  blockingErrorOptions,
  children,
}: CustomerTabShellProps) {
  const phase = resolveDirectoryLoadPhase(fetchState);

  if (phase === 'loading') {
    return <PageSkeleton />;
  }

  if (phase === 'blocking-error' && fetchState.error) {
    return panelError(fetchState.error, blockingErrorTitle, blockingErrorOptions);
  }

  return (
    <>
      {title ? <header >{title}</header> : null}
      {shouldShowDirectoryRefreshError(fetchState) && fetchState.error
        ? panelError(fetchState.error, refreshErrorTitle)
        : null}
      {children}
    </>
  );
}
