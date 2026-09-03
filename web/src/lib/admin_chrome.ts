/**
 * First-party admin chrome class strings (flat zinc). Use in @/components/ui only;
 * domains/shell import primitives, not these tokens directly.
 */
export const adminChrome = {
  control:
    'h-8 rounded-sm border border-zinc-200 bg-white px-3 text-sm text-zinc-900 transition-colors placeholder:text-zinc-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-zinc-400 focus-visible:ring-offset-2 focus-visible:ring-offset-white disabled:cursor-not-allowed disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100 dark:focus-visible:ring-offset-zinc-950',
  controlGhost:
    'h-8 rounded-sm border border-transparent bg-transparent px-3 text-sm text-zinc-900 transition-colors hover:bg-zinc-100 dark:text-zinc-100 dark:hover:bg-zinc-800',
  panel:
    'rounded-md border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-950',
  panelMuted: 'rounded-md bg-zinc-50 dark:bg-zinc-900',
  overlayBackdrop: 'fixed inset-0 z-50 bg-black/80',
  floating:
    'z-50 rounded-md border border-zinc-200 bg-white p-1 shadow-lg dark:border-zinc-800 dark:bg-zinc-950',
  menuItem:
    'relative flex w-full cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm text-zinc-900 outline-none hover:bg-zinc-100 disabled:pointer-events-none disabled:opacity-50 dark:text-zinc-100 dark:hover:bg-zinc-800',
  tableHead:
    'h-10 bg-zinc-50 px-3 text-left align-middle text-xs font-semibold text-zinc-500 dark:bg-zinc-900 dark:text-zinc-400',
  tableCell: 'px-3 py-2 align-middle text-sm text-zinc-900 dark:text-zinc-100',
  muted: 'text-zinc-500 dark:text-zinc-400',
  pageTitle: 'text-xl font-semibold tracking-tight text-zinc-950 dark:text-zinc-50',
} as const;

export type ButtonVariant = 'default' | 'secondary' | 'outline' | 'ghost' | 'destructive' | 'link';

export const buttonVariantClass: Record<ButtonVariant, string> = {
  default:
    'border-zinc-950 bg-zinc-950 text-white hover:bg-zinc-800 dark:border-zinc-100 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200',
  secondary:
    'border-transparent bg-zinc-100 text-zinc-900 hover:bg-zinc-200 dark:bg-zinc-800 dark:text-zinc-100 dark:hover:bg-zinc-700',
  outline:
    'border-zinc-200 bg-white text-zinc-700 hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-200 dark:hover:bg-zinc-900',
  ghost:
    'border-transparent bg-transparent text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800',
  destructive: 'border-red-600 bg-red-600 text-white hover:bg-red-700',
  link: 'border-0 bg-transparent text-zinc-900 underline-offset-4 hover:underline dark:text-zinc-100',
};

export type BadgeVariant = 'default' | 'secondary' | 'destructive' | 'outline';

export const badgeVariantClass: Record<BadgeVariant, string> = {
  default: 'border-transparent bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900',
  secondary: 'border-transparent bg-zinc-100 text-zinc-900 dark:bg-zinc-800 dark:text-zinc-100',
  destructive: 'border-transparent bg-red-600 text-white',
  outline: 'border-zinc-200 text-zinc-900 dark:border-zinc-700 dark:text-zinc-100',
};
