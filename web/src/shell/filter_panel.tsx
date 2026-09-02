import type { FormHTMLAttributes, HTMLAttributes, ReactNode } from 'react';

import { Label } from '@/components/ui/label';
import { cn } from '@/lib/utils';

const DIRECTORY_FILTER_GRID =
  'grid grid-cols-1 items-end gap-3 sm:grid-cols-2 xl:grid-cols-[minmax(0,1.4fr)_minmax(0,0.85fr)_minmax(0,1.1fr)_auto_auto_auto_auto]';

const AUTO_FILL_FILTER_GRID =
  'grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4';

export type FilterPanelProps = HTMLAttributes<HTMLElement> & {
  as?: 'section' | 'div';
};

export function FilterPanel({ as: Component = 'section', className, children, ...props }: FilterPanelProps) {
  return (
    <Component className={cn('ui-filter-panel', className)} {...props}>
      {children}
    </Component>
  );
}

export type DirectoryFilterFormProps = FormHTMLAttributes<HTMLFormElement> & {
  layout?: 'directory' | 'auto-fill';
};

export function DirectoryFilterForm({
  className,
  layout = 'auto-fill',
  ...props
}: DirectoryFilterFormProps) {
  return (
    <form
      className={cn(layout === 'directory' ? DIRECTORY_FILTER_GRID : AUTO_FILL_FILTER_GRID, className)}
      {...props}
    />
  );
}

export type FilterFieldProps = {
  children: ReactNode;
  className?: string;
  htmlFor?: string;
  label: string;
  wide?: boolean;
};

export function FilterField({ children, className, htmlFor, label, wide = false }: FilterFieldProps) {
  return (
    <div
      className={cn('grid min-w-0 gap-2', wide && 'sm:col-span-2 xl:col-span-1', className)}
    >
      <Label htmlFor={htmlFor}>{label}</Label>
      {children}
    </div>
  );
}
