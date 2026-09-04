/**
 * First-party admin chrome class strings. Use in @/components/ui only;
 * domains/shell import primitives, not these tokens directly.
 */
import { adminKit } from '@/lib/admin_kit';

export const adminChrome = {
  control: cnControl(),
  controlGhost:
    'min-h-7 rounded-[5px] border border-transparent bg-transparent px-2 py-1 text-[13px] leading-[18px] text-foreground transition-colors hover:bg-accent hover:text-accent-foreground',
  panel: 'rounded-[10px] border border-border bg-card text-card-foreground',
  panelMuted: 'rounded-[10px] bg-muted text-muted-foreground',
  overlayBackdrop: 'fixed inset-0 z-50 bg-black/80',
  floating:
    'z-50 rounded-[5px] border border-border bg-popover text-popover-foreground p-1 shadow-lg',
  menuItem:
    'relative flex w-full cursor-pointer select-none items-center whitespace-nowrap rounded-[5px] px-2 py-1.5 text-[13px] text-foreground outline-none hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50',
  tableHead:
    'h-[34px] bg-muted/50 px-4 text-left align-middle text-[11px] font-bold uppercase leading-[14px] text-muted-foreground',
  tableCell: 'px-4 py-0 align-middle text-[13px] leading-[18px] text-foreground',
  muted: 'text-muted-foreground',
  pageTitle: 'text-lg font-bold tracking-tight text-foreground',
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
  | 'secondary'
  | 'outline'
  | 'ghost'
  | 'destructive'
  | 'link';

export const buttonVariantClass: Record<ButtonVariant, string> = {
  default:
    'border-primary bg-primary text-primary-foreground hover:bg-primary/90',
  brand:
    'border-emerald-500 bg-emerald-500 text-slate-900 hover:border-emerald-600 hover:bg-emerald-600 dark:hover:border-emerald-400 dark:hover:bg-emerald-400',
  secondary:
    'border-border bg-secondary text-secondary-foreground hover:bg-secondary/80',
  outline:
    'border-border bg-background text-foreground hover:bg-accent hover:text-accent-foreground',
  ghost:
    'border-transparent bg-transparent text-muted-foreground hover:bg-accent hover:text-foreground',
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
  active: 'border-transparent bg-emerald-500/10 text-emerald-500',
  paused: 'border-transparent bg-amber-500/10 text-amber-500',
  archived: 'border-border bg-muted text-muted-foreground',
  error: 'border-transparent bg-destructive/10 text-destructive',
  draft: 'border-transparent bg-sky-500/10 text-sky-500',
  scheduled: 'border-transparent bg-orange-500/10 text-orange-500',
};
