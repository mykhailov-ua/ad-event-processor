import type { ReactNode } from 'react';

import { PageSkeleton } from '@/shell/page_skeleton';
import { panelError } from '@/shell/panel_error';
import {
  type DirectoryFetchState,
  resolveDirectoryLoadPhase,
  shouldShowDirectoryRefreshError,
} from '@/shell/directory_load_state';

export type EditorPageShellProps = {
  blockingErrorTitle: string;
  refreshErrorTitle?: string;
  fetchState: DirectoryFetchState;
  children: ReactNode;
};

export function EditorPageShell({
  blockingErrorTitle,
  refreshErrorTitle = 'Refresh failed',
  fetchState,
  children,
}: EditorPageShellProps) {
  const phase = resolveDirectoryLoadPhase(fetchState);

  if (phase === 'loading') {
    return <PageSkeleton />;
  }

  if (phase === 'blocking-error' && fetchState.error) {
    return panelError(fetchState.error, blockingErrorTitle);
  }

  return (
    <>
      {shouldShowDirectoryRefreshError(fetchState) && fetchState.error
        ? panelError(fetchState.error, refreshErrorTitle)
        : null}
      {children}
    </>
  );
}
