import type { ReactNode } from 'react';

import { SectionNav } from '@/shell/section_nav';
import { PageChrome } from '@/shell/page_chrome';
import { PageSkeleton } from '@/shell/page_skeleton';
import { AdminError } from '@/shell/admin_error';
import {
  type DirectoryFetchState,
  resolveDirectoryLoadPhase,
  shouldShowDirectoryRefreshError,
} from '@/shell/directory_load_state';
import type { SectionNavItem } from '@/lib/nav_config';

export const INTEGRATIONS_NAV_ITEMS: SectionNavItem[] = [
  { path: '/integrations', label: 'Hub', exact: true },
  { path: '/integrations/api-keys', label: 'Service accounts' },
  { path: '/integrations/cost-sync', label: 'Cost sync' },
  { path: '/integrations/postbacks', label: 'Postbacks' },
  { path: '/integrations/debugger', label: 'Debugger' },
  { path: '/integrations/schemas', label: 'Schemas' },
  { path: '/integrations/platform-campaigns', label: 'Platform links' },
  { path: '/integrations/affiliate-presets', label: 'Affiliate presets' },
];

export function IntegrationsNav() {
  return <SectionNav items={INTEGRATIONS_NAV_ITEMS} label="Integrations sections" />;
}

export function integrationsPanelError(error: Error, title: string) {
  return <AdminError error={error} title={title} />;
}

export type IntegrationsPageWithLoadProps = {
  title: ReactNode;
  blockingErrorTitle: string;
  refreshErrorTitle?: string;
  fetchState: DirectoryFetchState;
  /** Chrome below nav (e.g. customer scope) on every non-loading phase. */
  header?: ReactNode;
  /** Non-fetch errors (mutation, export) above main content. */
  alerts?: ReactNode;
  children: ReactNode;
};

export function IntegrationsPageWithLoad({
  title,
  blockingErrorTitle,
  refreshErrorTitle = 'Refresh failed',
  fetchState,
  header,
  alerts,
  children,
}: IntegrationsPageWithLoadProps) {
  const phase = resolveDirectoryLoadPhase(fetchState);

  if (phase === 'loading') {
    return <PageSkeleton />;
  }

  if (phase === 'blocking-error' && fetchState.error) {
    return (
      <PageChrome title={title}>
        <IntegrationsNav />
        {header}
        {integrationsPanelError(fetchState.error, blockingErrorTitle)}
      </PageChrome>
    );
  }

  return (
    <PageChrome title={title}>
      <IntegrationsNav />
      {header}
      {shouldShowDirectoryRefreshError(fetchState) && fetchState.error
        ? integrationsPanelError(fetchState.error, refreshErrorTitle)
        : null}
      {alerts}
      {children}
    </PageChrome>
  );
}
