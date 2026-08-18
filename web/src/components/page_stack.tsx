import type { ReactNode } from 'react';

export type PageStackProps = {
  children: ReactNode;

  gap?: 'sm' | 'md' | 'lg' | 'xl';
};

const GAP_CLASS: Record<NonNullable<PageStackProps['gap']>, string> = {
  sm: 'stack stack--sm',
  md: 'stack',
  lg: 'stack stack--lg',
  xl: 'stack stack--lg',
};

export function PageStack({ children, gap = 'lg' }: PageStackProps) {
  return <div className={GAP_CLASS[gap]}>{children}</div>;
}
