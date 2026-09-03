import * as React from 'react';

import { cn } from '@/lib/utils';

const TableFrame = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn('overflow-hidden rounded-md border border-zinc-200 bg-white p-0 dark:border-zinc-800 dark:bg-zinc-950', className)}
      {...props}
    />
  ),
);
TableFrame.displayName = 'TableFrame';

const Table = React.forwardRef<
  HTMLTableElement,
  React.HTMLAttributes<HTMLTableElement> & { bare?: boolean }
>(({ className, bare = false, ...props }, ref) => {
  const table = (
    <table ref={ref} className={cn('w-full caption-bottom border-collapse text-sm', className)} {...props} />
  );
  if (bare) {
    return table;
  }
  return <div className="relative w-full min-w-0 max-w-full overflow-x-auto">{table}</div>;
});
Table.displayName = 'Table';

const TableHeader = React.forwardRef<
  HTMLTableSectionElement,
  React.HTMLAttributes<HTMLTableSectionElement>
>(({ className, ...props }, ref) => (
  <thead ref={ref} className={cn('[&_tr]:border-b', className)} {...props} />
));
TableHeader.displayName = 'TableHeader';

const TableBody = React.forwardRef<
  HTMLTableSectionElement,
  React.HTMLAttributes<HTMLTableSectionElement>
>(({ className, ...props }, ref) => (
  <tbody ref={ref} className={cn('[&_tr:last-child]:border-0', className)} {...props} />
));
TableBody.displayName = 'TableBody';

const TableFooter = React.forwardRef<
  HTMLTableSectionElement,
  React.HTMLAttributes<HTMLTableSectionElement>
>(({ className, ...props }, ref) => (
  <tfoot
    ref={ref}
    className={cn(
      'border-t border-zinc-200 bg-zinc-50 font-medium dark:border-zinc-800 dark:bg-zinc-900 [&>tr]:last:border-b-0',
      className,
    )}
    {...props}
  />
));
TableFooter.displayName = 'TableFooter';

const TableRow = React.forwardRef<HTMLTableRowElement, React.HTMLAttributes<HTMLTableRowElement>>(
  ({ className, ...props }, ref) => (
    <tr
      ref={ref}
      className={cn(
        'border-b border-zinc-100 transition-colors hover:bg-zinc-50 data-[state=selected]:bg-blue-50 dark:border-zinc-800 dark:hover:bg-zinc-900/60 dark:data-[state=selected]:bg-blue-950/30',
        className,
      )}
      {...props}
    />
  ),
);
TableRow.displayName = 'TableRow';

const TableHead = React.forwardRef<
  HTMLTableCellElement,
  React.ThHTMLAttributes<HTMLTableCellElement>
>(({ className, ...props }, ref) => (
  <th
    ref={ref}
    className={cn(
      'h-10 bg-zinc-50 px-2 text-left align-middle text-xs font-semibold text-zinc-500 dark:bg-zinc-900 dark:text-zinc-400 [&:has([role=checkbox])]:pr-0 [&>[role=checkbox]]:translate-y-[2px]',
      className,
    )}
    {...props}
  />
));
TableHead.displayName = 'TableHead';

const TableCell = React.forwardRef<
  HTMLTableCellElement,
  React.TdHTMLAttributes<HTMLTableCellElement>
>(({ className, ...props }, ref) => (
  <td
    ref={ref}
    className={cn(
      'px-2 py-1.5 align-middle text-zinc-900 dark:text-zinc-100 [&:has([role=checkbox])]:pr-0 [&>[role=checkbox]]:translate-y-[2px]',
      className,
    )}
    {...props}
  />
));
TableCell.displayName = 'TableCell';

const TableCaption = React.forwardRef<
  HTMLTableCaptionElement,
  React.HTMLAttributes<HTMLTableCaptionElement>
>(({ className, ...props }, ref) => (
  <caption ref={ref} className={cn('mt-4 text-sm text-zinc-500 dark:text-zinc-400', className)} {...props} />
));
TableCaption.displayName = 'TableCaption';

export {
  TableFrame,
  Table,
  TableHeader,
  TableBody,
  TableFooter,
  TableHead,
  TableRow,
  TableCell,
  TableCaption,
};
