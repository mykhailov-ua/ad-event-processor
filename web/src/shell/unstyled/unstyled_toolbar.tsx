import type { ReactNode } from 'react';

export type UnstyledToolbarProps = {
  'aria-label'?: string;
  children: ReactNode;
  'data-testid'?: string;
};

export function UnstyledToolbar({
  'aria-label': ariaLabel = 'Actions',
  children,
  'data-testid': testId,
}: UnstyledToolbarProps) {
  return (
    <div aria-label={ariaLabel} data-role="toolbar" data-testid={testId} role="toolbar">
      {children}
    </div>
  );
}
