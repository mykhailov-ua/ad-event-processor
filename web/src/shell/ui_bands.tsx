import type { HTMLAttributes, ReactNode } from 'react';

import { adminKit } from '@/lib/admin_kit';
import { uiSurfaces } from '@/lib/ui_surfaces';
import { cn } from '@/lib/utils';

export type ToolbarBandProps = HTMLAttributes<HTMLDivElement> & {
  split?: boolean;
};

export function ToolbarBand({ split = false, className, children, ...props }: ToolbarBandProps) {
  return (
    <div
      className={cn(split ? uiSurfaces.toolbarBandSplit : uiSurfaces.toolbarBand, className)}
      {...props}
    >
      {children}
    </div>
  );
}

export function ToolbarBandActions({
  className,
  children,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn(uiSurfaces.toolbarBandActions, className)} role="toolbar" {...props}>
      {children}
    </div>
  );
}

export type ChipRowProps = HTMLAttributes<HTMLDivElement>;

export function ChipRow({ className, children, ...props }: ChipRowProps) {
  return (
    <div className={cn(uiSurfaces.chipRow, className)} {...props}>
      {children}
    </div>
  );
}

export type SummaryBandProps = HTMLAttributes<HTMLDivElement>;

export function SummaryBand({ className, children, ...props }: SummaryBandProps) {
  return (
    <div className={cn(uiSurfaces.summaryBand, adminKit.panelRadius, className)} {...props}>
      {children}
    </div>
  );
}

export function SummaryBandDivider() {
  return <span aria-hidden className={uiSurfaces.summaryBandDivider} />;
}

export type StatusMetricsBandProps = HTMLAttributes<HTMLDivElement>;

export function StatusMetricsBand({ className, children, ...props }: StatusMetricsBandProps) {
  return (
    <div className={cn(uiSurfaces.statusMetricsBand, className)} {...props}>
      {children}
    </div>
  );
}

export type TableHostProps = HTMLAttributes<HTMLDivElement> & {
  /** Stretch table chrome to fill remaining viewport height (rare; default hugs row content). */
  fill?: boolean;
};

export function TableHost({ fill = false, className, children, ...props }: TableHostProps) {
  return (
    <div
      className={cn(
        fill ? uiSurfaces.tableHostFill : uiSurfaces.tableHost,
        adminKit.panelRadius,
        className
      )}
      {...props}
    >
      {children}
    </div>
  );
}

export type TableHostScrollProps = HTMLAttributes<HTMLDivElement>;

export function TableHostScroll({ className, children, ...props }: TableHostScrollProps) {
  return (
    <div className={cn(uiSurfaces.tableHostScroll, className)} {...props}>
      {children}
    </div>
  );
}

export type DirectoryStackProps = HTMLAttributes<HTMLDivElement>;

export function DirectoryStack({ className, children, ...props }: DirectoryStackProps) {
  return (
    <div className={cn(uiSurfaces.directoryStack, className)} {...props}>
      {children}
    </div>
  );
}

export type MetaLinksBandProps = HTMLAttributes<HTMLDivElement>;

export function MetaLinksBand({ className, children, ...props }: MetaLinksBandProps) {
  return (
    <div className={cn(uiSurfaces.metaLinksBand, className)} {...props}>
      {children}
    </div>
  );
}

export type ActionLinksBandProps = HTMLAttributes<HTMLDivElement>;

export function ActionLinksBand({ className, children, ...props }: ActionLinksBandProps) {
  return (
    <div className={cn(uiSurfaces.actionLinksBand, className)} {...props}>
      {children}
    </div>
  );
}
