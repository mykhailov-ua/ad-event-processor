import type { FormHTMLAttributes, HTMLAttributes, ReactNode } from 'react';

import { Label } from '@/components/ui/label';
import { adminKit } from '@/lib/admin_kit';
import { uiSurfaces } from '@/lib/ui_surfaces';
import { cn } from '@/lib/utils';

import {
  AUTO_FILL_FILTER_GRID,
  CAMPAIGNS_FILTER_ROW,
  COMPACT_TOOLBAR_ROW_CLASS,
  DIRECTORY_CONTENT_BAND_CLASS,
  DIRECTORY_FIELD_LABEL_CLASS,
  DIRECTORY_FILTER_FORM_STACK_CLASS,
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
  DIRECTORY_CONTENT_BAND_CLASS,
  DIRECTORY_FIELD_LABEL_CLASS,
  DIRECTORY_FILTER_FORM_STACK_CLASS,
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
    <Component className={cn(uiSurfaces.filterPanel, className)} {...props}>
      {children}
    </Component>
  );
}

export type DirectoryFilterFormProps = FormHTMLAttributes<HTMLFormElement> & {
  layout?: 'directory' | 'auto-fill' | 'campaigns';
};

export function DirectoryFilterForm({
  layout = 'auto-fill',
  className,
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
  htmlFor?: string;
  label: string;
  wide?: boolean;
};

export function FilterField({
  children,
  htmlFor,
  label,
  wide = false,
}: FilterFieldProps) {
  return (
    <div className={cn('grid min-w-0', adminKit.fieldLabelGap, wide && 'md:col-span-2')}>
      <Label className={DIRECTORY_FIELD_LABEL_CLASS} htmlFor={htmlFor}>
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
  return <div className={cn(uiSurfaces.toolbarBand, className)}>{children}</div>;
}
