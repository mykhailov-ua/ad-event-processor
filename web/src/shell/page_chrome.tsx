import type { ReactNode } from 'react';

import { PageLayout } from '@/shell/page_layout';

export type PageChromeProps = {
  title: ReactNode;
  description?: ReactNode;
  badge?: ReactNode;
  actions?: ReactNode;
  controlPanel?: ReactNode;
  aside?: ReactNode;
  footer?: ReactNode;
  workspaceClassName?: string;
  children?: ReactNode;
};

export function PageChrome({
  title,
  description,
  badge,
  actions,
  controlPanel,
  aside,
  footer,
  workspaceClassName,
  children,
}: PageChromeProps) {
  return (
    <PageLayout
      aside={aside}
      badge={badge}
      controlPanel={controlPanel}
      description={description}
      footer={footer}
      headerActions={actions}
      title={title}
      workspaceClassName={workspaceClassName}
    >
      {children}
    </PageLayout>
  );
}
