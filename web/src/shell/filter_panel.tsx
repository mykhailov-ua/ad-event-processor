import type { FormHTMLAttributes, HTMLAttributes, ReactNode } from 'react';

import { Label } from '@/components/ui/label';
import { cn } from '@/lib/utils';

const DIRECTORY_FILTER_GRID =
  'grid grid-cols-[repeat(auto-fill,11rem)] items-end gap-x-3 gap-y-3';

const CAMPAIGNS_FILTER_ROW =
  'flex w-full flex-nowrap items-start gap-2 [&>.admin-campaigns-filter-field]:min-w-0 [&>.admin-campaigns-filter-field]:flex-1 [&>.admin-campaigns-filter-field]:basis-0';

const AUTO_FILL_FILTER_GRID =
  'grid grid-cols-[repeat(auto-fill,12rem)] items-end gap-x-4 gap-y-4';

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
  layout?: 'directory' | 'auto-fill' | 'campaigns';
};

export function DirectoryFilterForm({
  className,
  layout = 'auto-fill',
  ...props
}: DirectoryFilterFormProps) {
  const layoutClass =
    layout === 'campaigns'
      ? CAMPAIGNS_FILTER_ROW
      : layout === 'directory'
        ? DIRECTORY_FILTER_GRID
        : AUTO_FILL_FILTER_GRID;

  return <form className={cn(layoutClass, className)} {...props} />;
}

export type FilterFieldProps = {
  children: ReactNode;
  className?: string;
  htmlFor?: string;
  label: string;
  labelClassName?: string;
  wide?: boolean;
};

export function FilterField({
  children,
  className,
  htmlFor,
  label,
  labelClassName,
  wide = false,
}: FilterFieldProps) {
  return (
    <div
      className={cn('grid w-full min-w-0 gap-1.5', wide && 'sm:col-span-2 xl:col-span-1', className)}
    >
      <Label className={labelClassName} htmlFor={htmlFor}>{label}</Label>
      {children}
    </div>
  );
}
