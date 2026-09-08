import type { FormHTMLAttributes, HTMLAttributes, ReactNode } from 'react';

import { Label } from '@/components/ui/label';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

import {
  AUTO_FILL_FILTER_GRID,
  CAMPAIGNS_FILTER_ROW,
  COMPACT_TOOLBAR_ROW_CLASS,
  DIRECTORY_FILTER_GRID,
  EDITOR_MAIN_COLUMN_CLASS,
  FILTER_PANEL_FLAT_CLASS,
  FILTER_PANEL_NARROW_CLASS,
  FILTER_PANEL_SUMMARY_CLASS,
  FILTER_PANEL_WIDE_CLASS,
  INLINE_FILTER_ACTION_GRID_CLASS,
  INLINE_FILTER_ACTION_GRID_THREE_ACTIONS_CLASS,
  INLINE_FILTER_ACTION_GRID_TWO_ACTIONS_CLASS,
  INLINE_FILTER_ACTION_GRID_TWO_FIELDS_CLASS,
  INLINE_FILTER_ACTION_GRID_WIDE_CLASS,
  NARROW_EDIT_FORM_CLASS,
  NARROW_EDIT_FORM_WIDE_CLASS,
  SECTION_PANEL_CLASS,
  SECTION_SURFACE_RAISED_CLASS,
} from '@/shell/filter_panel_classes';

export {
  AUTO_FILL_FILTER_GRID,
  CAMPAIGNS_FILTER_ROW,
  COMPACT_TOOLBAR_ROW_CLASS,
  DIRECTORY_FILTER_GRID,
  EDITOR_MAIN_COLUMN_CLASS,
  FILTER_PANEL_FLAT_CLASS,
  FILTER_PANEL_NARROW_CLASS,
  FILTER_PANEL_SUMMARY_CLASS,
  FILTER_PANEL_WIDE_CLASS,
  INLINE_FILTER_ACTION_GRID_CLASS,
  INLINE_FILTER_ACTION_GRID_THREE_ACTIONS_CLASS,
  INLINE_FILTER_ACTION_GRID_TWO_ACTIONS_CLASS,
  INLINE_FILTER_ACTION_GRID_TWO_FIELDS_CLASS,
  INLINE_FILTER_ACTION_GRID_WIDE_CLASS,
  NARROW_EDIT_FORM_CLASS,
  NARROW_EDIT_FORM_WIDE_CLASS,
  SECTION_PANEL_CLASS,
  SECTION_SURFACE_RAISED_CLASS,
};

export type FilterPanelProps = HTMLAttributes<HTMLElement> & {
  as?: 'section' | 'div';
};

export function FilterPanel({
  as: Component = 'section',
  className,
  children,
  ...props
}: FilterPanelProps) {
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
      className={cn(
        'grid w-full min-w-0',
        adminKit.fieldLabelGap,
        wide && 'sm:col-span-2 xl:col-span-1',
        className
      )}
    >
      <Label className={labelClassName} htmlFor={htmlFor}>
        {label}
      </Label>
      {children}
    </div>
  );
}

/** Full-width action row under filter fields; keeps buttons start-aligned, not centered in grid cells. */
export function FilterFormActions({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn('col-span-full flex flex-wrap items-center justify-start gap-2', className)}>
      {children}
    </div>
  );
}
