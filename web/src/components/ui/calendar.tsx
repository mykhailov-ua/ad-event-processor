import * as React from 'react';
import { ChevronDown, ChevronLeft, ChevronRight } from 'lucide-react';
import { DayButton, DayPicker, getDefaultClassNames } from 'react-day-picker';

import { cn } from '@/lib/utils';
import { Button, buttonVariants } from '@/components/ui/button';

export type CalendarProps = React.ComponentProps<typeof DayPicker> & {
  buttonVariant?: React.ComponentProps<typeof Button>['variant'];
  variant?: 'default' | 'admin';
};

function Calendar({
  className,
  classNames,
  showOutsideDays = true,
  captionLayout = 'label',
  buttonVariant = 'ghost',
  variant = 'default',
  formatters,
  components,
  ...props
}: CalendarProps) {
  const defaultClassNames = getDefaultClassNames();
  const isAdmin = variant === 'admin';

  return (
    <DayPicker
      showOutsideDays={showOutsideDays}
      className={cn(
        isAdmin
          ? 'group/calendar [--cell-size:1.875rem] bg-transparent p-0'
          : 'group/calendar p-3 [--cell-size:2rem] bg-transparent [[data-slot=card-content]_&]:bg-transparent [[data-slot=popover-content]_&]:bg-transparent',
        String.raw`rtl:**:[.rdp-button\_next>svg]:rotate-180`,
        String.raw`rtl:**:[.rdp-button\_previous>svg]:rotate-180`,
        className,
      )}
      captionLayout={captionLayout}
      formatters={{
        formatMonthDropdown: (date) => date.toLocaleString('default', { month: 'short' }),
        ...formatters,
      }}
      classNames={{
        root: cn('w-fit', defaultClassNames.root),
        months: cn('relative flex flex-col gap-4 md:flex-row', defaultClassNames.months),
        month: cn('flex w-full flex-col gap-4', defaultClassNames.month),
        nav: cn(
          'absolute inset-x-0 top-0 flex w-full items-center justify-between gap-1',
          defaultClassNames.nav,
        ),
        button_previous: cn(
          isAdmin
            ? 'inline-flex h-8 items-center justify-center gap-2 rounded-md border border-zinc-200 bg-white px-3 text-sm font-medium transition active:scale-[0.98] disabled:pointer-events-none disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-900 h-8 w-8 p-0 h-[--cell-size] w-[--cell-size] min-h-0 p-0 aria-disabled:opacity-50'
            : buttonVariants({ variant: buttonVariant, shape: 'square' }),
          'h-[--cell-size] w-[--cell-size] select-none',
          defaultClassNames.button_previous,
        ),
        button_next: cn(
          isAdmin
            ? 'inline-flex h-8 items-center justify-center gap-2 rounded-md border border-zinc-200 bg-white px-3 text-sm font-medium transition active:scale-[0.98] disabled:pointer-events-none disabled:opacity-50 dark:border-zinc-700 dark:bg-zinc-900 h-8 w-8 p-0 h-[--cell-size] w-[--cell-size] min-h-0 p-0 aria-disabled:opacity-50'
            : buttonVariants({ variant: buttonVariant, shape: 'square' }),
          'h-[--cell-size] w-[--cell-size] select-none',
          defaultClassNames.button_next,
        ),
        month_caption: cn(
          'flex h-[--cell-size] w-full items-center justify-center px-[--cell-size]',
          isAdmin && 'text-admin-dense font-semibold text-zinc-900 dark:text-zinc-50',
          defaultClassNames.month_caption,
        ),
        dropdowns: cn(
          'flex h-[--cell-size] w-full items-center justify-center gap-1.5 text-sm font-medium',
          defaultClassNames.dropdowns,
        ),
        dropdown_root: cn(
          isAdmin
            ? 'has-focus:border-blue-600 relative rounded-sm border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-950'
            : 'has-focus:border-ring border-input has-focus:ring-ring/50 has-focus:ring-[3px] relative rounded-xl border border-border/50',
          defaultClassNames.dropdown_root,
        ),
        dropdown: cn(
          isAdmin ? 'bg-zinc-50 dark:bg-zinc-900 absolute inset-0 opacity-0' : 'bg-popover absolute inset-0 opacity-0',
          defaultClassNames.dropdown,
        ),
        caption_label: cn(
          'select-none font-medium',
          captionLayout === 'label'
            ? isAdmin
              ? 'text-admin-dense font-semibold text-zinc-900 dark:text-zinc-50'
              : 'text-sm'
            : '[&>svg]:text-muted-foreground flex h-8 items-center gap-1 rounded-md pl-2 pr-1 text-sm [&>svg]:size-3.5',
          defaultClassNames.caption_label,
        ),
        month_grid: cn('w-full border-collapse', defaultClassNames.month_grid),
        weekdays: cn('flex', defaultClassNames.weekdays),
        weekday: cn(
          isAdmin
            ? 'flex-1 select-none rounded-sm text-xs font-semibold text-zinc-500 dark:text-zinc-400'
            : 'text-muted-foreground flex-1 select-none rounded-lg text-xs font-normal',
          defaultClassNames.weekday,
        ),
        week: cn('mt-2 flex w-full gap-1', defaultClassNames.week),
        week_number_header: cn('w-[--cell-size] select-none', defaultClassNames.week_number_header),
        week_number: cn(
          'text-muted-foreground select-none text-xs',
          defaultClassNames.week_number,
        ),
        day: cn(
          'group/day relative flex flex-1 items-center justify-center p-0.5 text-center',
          defaultClassNames.day,
        ),
        range_start: cn(defaultClassNames.range_start),
        range_middle: cn(defaultClassNames.range_middle),
        range_end: cn(defaultClassNames.range_end),
        today: cn(
          isAdmin
            ? 'rounded-sm bg-zinc-100 dark:bg-zinc-800 font-semibold text-zinc-900 dark:text-zinc-50 data-[selected=true]:rounded-sm'
            : 'rounded-lg bg-accent/80 text-accent-foreground data-[selected=true]:rounded-lg',
          defaultClassNames.today,
        ),
        outside: cn(
          isAdmin
            ? 'text-zinc-500 dark:text-zinc-400 opacity-40 aria-selected:opacity-40'
            : 'text-muted-foreground opacity-40 aria-selected:opacity-40',
          defaultClassNames.outside,
        ),
        disabled: cn(
          isAdmin ? 'text-zinc-500 dark:text-zinc-400 opacity-50' : 'text-muted-foreground opacity-50',
          defaultClassNames.disabled,
        ),
        hidden: cn('invisible', defaultClassNames.hidden),
        ...classNames,
      }}
      components={{
        Root: ({ className: rootClassName, rootRef, ...rootProps }) => (
          <div data-slot="calendar" ref={rootRef} className={cn(rootClassName)} {...rootProps} />
        ),
        Chevron: ({ className: chevronClassName, orientation, ...chevronProps }) => {
          if (orientation === 'left') {
            return <ChevronLeft className={cn('size-4', chevronClassName)} {...chevronProps} />;
          }
          if (orientation === 'right') {
            return <ChevronRight className={cn('size-4', chevronClassName)} {...chevronProps} />;
          }
          return <ChevronDown className={cn('size-4', chevronClassName)} {...chevronProps} />;
        },
        DayButton: (dayButtonProps) => (
          <CalendarDayButton {...dayButtonProps} variant={variant} />
        ),
        WeekNumber: ({ children, ...weekProps }) => (
          <td {...weekProps}>
            <div className="flex size-[--cell-size] items-center justify-center text-center">
              {children}
            </div>
          </td>
        ),
        ...components,
      }}
      {...props}
    />
  );
}
Calendar.displayName = 'Calendar';

function CalendarDayButton({
  className,
  day,
  modifiers,
  variant = 'default',
  ...props
}: React.ComponentProps<typeof DayButton> & { variant?: 'default' | 'admin' }) {
  const defaultClassNames = getDefaultClassNames();
  const ref = React.useRef<HTMLButtonElement>(null);
  const isAdmin = variant === 'admin';

  React.useEffect(() => {
    if (modifiers.focused) {
      ref.current?.focus();
    }
  }, [modifiers.focused]);

  return (
    <Button
      ref={ref}
      variant="ghost"
      size="icon"
      data-day={day.date.toLocaleDateString()}
      data-selected-single={
        modifiers.selected &&
        !modifiers.range_start &&
        !modifiers.range_end &&
        !modifiers.range_middle
      }
      data-range-start={modifiers.range_start}
      data-range-end={modifiers.range_end}
      data-range-middle={modifiers.range_middle}
      className={cn(
        'size-[calc(var(--cell-size)-0.3rem)] p-0 font-normal leading-none',
        isAdmin
          ? 'min-h-0 border border-transparent bg-transparent shadow-none hover:bg-zinc-100 dark:bg-zinc-800 rounded-sm data-[selected-single=true]:border-blue-600 data-[selected-single=true]:bg-blue-600 data-[selected-single=true]:text-white data-[range-start=true]:border-blue-600 data-[range-start=true]:bg-blue-600 data-[range-start=true]:text-white data-[range-end=true]:border-blue-600 data-[range-end=true]:bg-blue-600 data-[range-end=true]:text-white data-[range-middle=true]:border-transparent data-[range-middle=true]:bg-blue-50 dark:bg-blue-950/40 data-[range-middle=true]:text-zinc-900 dark:text-zinc-100 group-data-[focused=true]/day:relative group-data-[focused=true]/day:z-10 group-data-[focused=true]/day:border-blue-600 group-data-[focused=true]/day:ring-0'
          : 'rounded-[var(--picker-radius-control,var(--radius))] data-[selected-single=true]:bg-primary data-[selected-single=true]:text-primary-foreground data-[range-start=true]:bg-primary data-[range-start=true]:text-primary-foreground data-[range-end=true]:bg-primary data-[range-end=true]:text-primary-foreground data-[range-middle=true]:bg-accent/55 data-[range-middle=true]:text-foreground group-data-[focused=true]/day:relative group-data-[focused=true]/day:z-10 group-data-[focused=true]/day:border-ring group-data-[focused=true]/day:ring-ring/50 group-data-[focused=true]/day:ring-[3px]',
        'flex items-center justify-center',
        '[&>span]:text-xs [&>span]:opacity-70',
        modifiers.outside &&
          (isAdmin ? 'text-zinc-500 dark:text-zinc-400 opacity-40' : 'text-muted-foreground opacity-40'),
        defaultClassNames.day,
        className,
      )}
      {...props}
    />
  );
}

export { Calendar, CalendarDayButton };
