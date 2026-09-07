/**
 * First-party admin chrome class strings. Use in @/components/ui only;
 * domains/shell import primitives, not these tokens directly.
 */
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export const adminChrome = {
  control: cnControl(),
  controlFieldGroup: cn(cnControl(), 'flex items-center gap-2'),
  controlFieldInset:
    'min-w-0 flex-1 border-0 bg-transparent p-0 text-foreground shadow-none outline-none placeholder:text-muted-foreground focus-visible:ring-0 focus-visible:ring-offset-0 disabled:cursor-not-allowed disabled:opacity-50',
  controlGhost: cn(
    adminKit.buttonShell,
    adminKit.controlRadius,
    'border border-transparent bg-transparent px-2 text-foreground transition-colors hover:bg-accent hover:text-accent-foreground'
  ),
  panel: cn(adminKit.panelRadius, 'border border-border bg-card text-card-foreground'),
  panelMuted: cn(adminKit.panelRadius, 'bg-muted text-muted-foreground'),
  overlayBackdrop: 'fixed inset-0 z-50 bg-foreground/20 dark:bg-background/75',
  floating: cn(
    'z-50 border border-border bg-popover text-popover-foreground p-1 shadow-lg',
    adminKit.controlRadius
  ),
  menuList: 'flex flex-col gap-1',
  menuItem: cn(
    'relative flex w-full cursor-pointer select-none items-center whitespace-nowrap px-2 py-1.5 text-[13px] text-foreground outline-none hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50',
    adminKit.controlRadius
  ),
  tableHead:
    'h-[34px] bg-muted/50 px-4 text-left align-middle text-[11px] font-normal uppercase leading-[14px] text-muted-foreground',
  tableCell: 'px-4 py-0 align-middle text-[13px] leading-[18px] text-foreground',
  muted: 'text-muted-foreground',
  pageTitle: 'text-lg font-normal tracking-tight text-foreground',
} as const;

function cnControl(): string {
  return [
    adminKit.controlHeight,
    adminKit.controlRadius,
    adminKit.controlText,
    'border border-input bg-background px-2 py-1 text-foreground transition-colors',
    'placeholder:text-muted-foreground',
    adminKit.focusRing,
    'disabled:cursor-not-allowed disabled:opacity-50',
    'aria-[invalid=true]:border-destructive aria-[invalid=true]:ring-destructive/30',
  ].join(' ');
}

export type ButtonVariant =
  | 'default'
  | 'brand'
  | 'accent'
  | 'secondary'
  | 'outline'
  | 'ghost'
  | 'destructive'
  | 'link';

export const buttonVariantClass: Record<ButtonVariant, string> = {
  default: 'border-primary bg-primary text-primary-foreground hover:bg-primary/90',
  brand:
    'border-admin-brand bg-admin-brand text-admin-brand-foreground hover:border-admin-brand-hover hover:bg-admin-brand-hover',
  accent:
    'border-chart-1/40 bg-chart-1/12 text-chart-1 hover:border-chart-1/60 hover:bg-chart-1/20',
  secondary: 'border-border bg-secondary text-secondary-foreground hover:bg-secondary/80',
  outline:
    'border-primary/25 bg-background text-foreground hover:border-primary/40 hover:bg-primary/5 hover:text-primary',
  ghost:
    'border-transparent bg-transparent text-muted-foreground hover:bg-primary/10 hover:text-primary',
  destructive:
    'border-destructive bg-destructive text-destructive-foreground hover:bg-destructive/90',
  link: 'border-0 bg-transparent text-primary underline-offset-4 hover:underline',
};

export type BadgeVariant =
  | 'default'
  | 'secondary'
  | 'destructive'
  | 'outline'
  | 'active'
  | 'paused'
  | 'archived'
  | 'error'
  | 'draft'
  | 'scheduled';

export const badgeVariantClass: Record<BadgeVariant, string> = {
  default: 'border-transparent bg-primary text-primary-foreground',
  secondary: 'border-transparent bg-secondary text-secondary-foreground',
  destructive: 'border-transparent bg-destructive/10 text-destructive',
  outline: 'border-border text-foreground',
  active: 'border-transparent bg-admin-status-active/10 text-admin-status-active',
  paused: 'border-transparent bg-admin-status-paused/10 text-admin-status-paused',
  archived: 'border-border bg-muted text-muted-foreground',
  error: 'border-transparent bg-destructive/10 text-destructive',
  draft: 'border-transparent bg-admin-status-draft/10 text-admin-status-draft',
  scheduled: 'border-transparent bg-admin-status-scheduled/10 text-admin-status-scheduled',
};
