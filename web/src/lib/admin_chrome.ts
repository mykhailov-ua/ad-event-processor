/**
 * First-party admin chrome class strings. Use in @/components/ui only;
 * domains/shell import primitives, not these tokens directly.
 */
import { adminKit } from '@/lib/admin_kit';
import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { cn } from '@/lib/utils';

export const adminChrome = {
  control: cnControl(),
  controlFieldGroup: cn(
    cnControl(),
    `flex items-center ${adminSpacing.gap.md} focus-within:border-border/40 focus-within:ring-0`
  ),
  controlFieldInset:
    'min-w-0 flex-1 border-0 bg-transparent p-0 text-foreground shadow-none outline-none placeholder:text-muted-foreground focus-visible:ring-0 focus-visible:ring-offset-0 disabled:cursor-not-allowed disabled:opacity-50',
  controlGhost: cn(
    adminKit.buttonShell,
    adminKit.controlRadius,
    `border border-transparent bg-transparent ${adminKit.controlPaddingX} text-foreground transition-colors hover:border-border hover:bg-accent hover:text-accent-foreground`
  ),
  panel: cn(adminKit.panelRadius, 'border border-border bg-card text-card-foreground'),
  panelMuted: cn(adminKit.panelRadius, 'bg-muted text-muted-foreground'),
  overlayBackdrop: 'fixed inset-0 z-50 bg-foreground/20 dark:bg-background/75',
  floating: cn(
    `z-50 border border-border bg-popover p-1 text-popover-foreground shadow-md shadow-black/10`,
    adminKit.controlRadius
  ),
  menuList: `flex flex-col ${adminSpacing.gap.xs} p-0.5`,
  menuItem: cn(
    `relative flex w-full cursor-pointer select-none items-center whitespace-nowrap ${adminKit.controlPaddingX} py-1.5 ${adminTypography.body} outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:bg-accent disabled:pointer-events-none disabled:opacity-60`,
    adminKit.nestedRadius
  ),
  menuItemSelected: 'bg-admin-selection text-foreground hover:bg-admin-selection',
  tableHead: `h-[34px] bg-muted/50 ${adminSpacing.inset.tableCellX} text-left align-middle ${adminTypography.tableHeader}`,
  tableCell: `${adminSpacing.inset.tableCellX} py-0 align-middle ${adminTypography.tableBody}`,
  muted: 'text-muted-foreground',
  pageTitle: adminTypography.pageTitle,
} as const;

function cnControl(): string {
  return [
    'min-h-7 h-auto',
    adminKit.controlRadius,
    adminKit.controlText,
    `${adminKit.controlBorder} bg-admin-control ${adminKit.controlPaddingX} py-1 text-foreground shadow-none transition-colors`,
    'placeholder:text-muted-foreground',
    'hover:border-border/40 focus-visible:border-border/40 focus-visible:outline-none focus-visible:ring-0',
    'disabled:cursor-not-allowed disabled:border-border/40 disabled:bg-admin-input-disabled disabled:text-muted-foreground disabled:opacity-100',
    'aria-[invalid=true]:border-destructive/50 aria-[invalid=true]:ring-0',
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
  | 'link'
  | 'headerHelp';

export const buttonVariantClass: Record<ButtonVariant, string> = {
  default:
    'border-admin-brand bg-admin-brand text-admin-brand-foreground shadow-none hover:border-admin-brand-hover hover:bg-admin-brand-hover',
  brand:
    'border-admin-brand bg-admin-brand text-admin-brand-foreground shadow-none hover:border-admin-brand-hover hover:bg-admin-brand-hover',
  accent:
    'border-primary/35 bg-admin-selection text-foreground shadow-none hover:border-primary/50 hover:bg-admin-selection',
  secondary:
    'border-border bg-accent text-foreground shadow-none hover:border-border hover:bg-muted',
  outline:
    'border-border/40 bg-admin-control text-foreground shadow-none hover:border-border/40 hover:bg-admin-control hover:text-foreground focus-visible:ring-0 focus-visible:border-border/40',
  ghost:
    'border-transparent bg-transparent text-muted-foreground shadow-none hover:border-transparent hover:bg-accent hover:text-foreground',
  destructive:
    'border-destructive bg-destructive text-destructive-foreground shadow-none hover:border-destructive hover:bg-destructive/90',
  link: 'border-0 bg-transparent text-primary underline-offset-4 shadow-none hover:underline',
  headerHelp:
    'border-transparent bg-admin-header-faq text-primary-foreground shadow-none hover:border-transparent hover:bg-admin-header-faq-hover',
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
