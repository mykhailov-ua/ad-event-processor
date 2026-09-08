import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

/** Campaign list reference: icon-only controls inside directory table body cells. */
const directoryTableRowIconButtonBaseClass = cn(
  'inline-flex shrink-0 items-center justify-center border border-transparent p-0',
  adminKit.controlRadius,
  'text-muted-foreground transition-colors active:scale-100',
  'hover:border-border hover:bg-primary/10 hover:text-primary',
  adminKit.focusRing,
  'disabled:pointer-events-none disabled:opacity-50'
);

export const directoryTableRowCopyButtonClass = cn(directoryTableRowIconButtonBaseClass, 'size-6');

export const directoryTableRowMenuButtonClass = cn(directoryTableRowIconButtonBaseClass, 'size-7');
