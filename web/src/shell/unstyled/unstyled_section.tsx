import type { ReactNode } from 'react';

export type UnstyledSectionProps = {
  title?: string;
  'aria-label'?: string;
  children: ReactNode;
  'data-testid'?: string;
};

export function UnstyledSection({
  title,
  'aria-label': ariaLabel,
  children,
  'data-testid': testId,
}: UnstyledSectionProps) {
  return (
    <section aria-label={ariaLabel} data-role="section" data-testid={testId}>
      {title ? <h2 data-role="section-title">{title}</h2> : null}
      {children}
    </section>
  );
}
