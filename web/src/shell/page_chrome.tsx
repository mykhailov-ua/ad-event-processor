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
  headerClassName?: string;
  mainClassName?: string;
  /** When omitted, footer enables viewport fill (directory tables). */
  fillViewport?: boolean;
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
  headerClassName,
  mainClassName = 'min-w-0 w-full',
  fillViewport,
  children,
}: PageChromeProps) {
  return (
    <PageLayout
      aside={aside}
      badge={badge}
      controlPanel={controlPanel}
      description={description}
      fillViewport={fillViewport}
      footer={footer}
      headerActions={actions}
      headerClassName={headerClassName}
      mainClassName={mainClassName}
      title={title}
      workspaceClassName={workspaceClassName}
    >
      {children}
    </PageLayout>
  );
}
