export type DirectoryFetchState = {
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

export type DirectoryLoadPhase = 'loading' | 'blocking-error' | 'ready';

export function resolveDirectoryLoadPhase(state: DirectoryFetchState): DirectoryLoadPhase {
  if (state.fetching && !state.hasSnapshot && !state.error) {
    return 'loading';
  }
  if (state.error && !state.hasSnapshot) {
    return 'blocking-error';
  }
  return 'ready';
}

export function shouldShowDirectoryRefreshError(state: DirectoryFetchState): boolean {
  return state.error != null && state.hasSnapshot;
}
