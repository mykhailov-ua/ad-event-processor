import { getDefaultClassNames } from 'react-day-picker';

import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

type CalendarCaptionLayout = 'label' | 'dropdown' | 'dropdown-months' | 'dropdown-years';

const NAV_BUTTON_CLASS = cn(
  adminKit.buttonShell,
  adminKit.nestedRadius,
  'h-8 w-8 shrink-0 border-border/50 bg-admin-control p-0 text-foreground/70 hover:border-border/50 hover:bg-accent hover:text-foreground disabled:pointer-events-none disabled:opacity-50'
);

export function buildCalendarClassNames(captionLayout: CalendarCaptionLayout = 'label') {
  const defaultClassNames = getDefaultClassNames();

  return {
    root: cn('w-fit', defaultClassNames.root),
    months: cn('relative flex flex-col gap-4 md:flex-row', defaultClassNames.months),
    month: cn('flex w-full flex-col gap-3', defaultClassNames.month),
    nav: cn(
      'absolute inset-x-0 top-0 flex w-full items-center justify-between',
      defaultClassNames.nav
    ),
    button_previous: cn(NAV_BUTTON_CLASS, defaultClassNames.button_previous),
    button_next: cn(NAV_BUTTON_CLASS, defaultClassNames.button_next),
    month_caption: cn(
      'flex h-8 w-full items-center justify-center px-8',
      defaultClassNames.month_caption
    ),
    dropdowns: cn(
      'flex h-8 w-full items-center justify-center gap-1.5 text-sm font-medium',
      defaultClassNames.dropdowns
    ),
    dropdown_root: cn(
      'relative border border-border/40 bg-admin-control has-[:focus]:border-border/40 has-[:focus]:ring-0',
      adminKit.nestedRadius,
      defaultClassNames.dropdown_root
    ),
    dropdown: cn('absolute inset-0 bg-transparent opacity-0', defaultClassNames.dropdown),
    caption_label: cn(
      'select-none font-semibold text-foreground',
      captionLayout === 'label'
        ? 'text-sm'
        : cn(
            'flex h-8 items-center gap-1 pl-2 pr-1 text-sm [&>svg]:size-3.5 [&>svg]:text-muted-foreground',
            adminKit.nestedRadius
          ),
      defaultClassNames.caption_label
    ),
    month_grid: cn('w-full border-collapse', defaultClassNames.month_grid),
    weekdays: cn('flex gap-1', defaultClassNames.weekdays),
    weekday: cn(
      'flex size-[var(--cell-size)] shrink-0 select-none items-center justify-center text-[11px] font-semibold uppercase leading-[14px] text-muted-foreground',
      defaultClassNames.weekday
    ),
    week: cn('mt-1 flex gap-1', defaultClassNames.week),
    week_number_header: cn('w-8 select-none', defaultClassNames.week_number_header),
    week_number: cn('select-none text-xs text-muted-foreground', defaultClassNames.week_number),
    day: cn(
      'group/day relative flex size-[var(--cell-size)] shrink-0 select-none overflow-hidden rounded-[4px] p-0 text-center',
      defaultClassNames.day
    ),
    range_start: cn('rounded-l-[8px]', defaultClassNames.range_start),
    range_middle: cn('rounded-none', defaultClassNames.range_middle),
    range_end: cn('rounded-r-[8px]', defaultClassNames.range_end),
    selected: '',
    today: '',
    focused: '',
    outside: cn(
      'text-muted-foreground aria-selected:text-muted-foreground',
      defaultClassNames.outside
    ),
    disabled: cn('text-muted-foreground opacity-40', defaultClassNames.disabled),
    hidden: cn('invisible', defaultClassNames.hidden),
  };
}

export const calendarRootClass = cn(
  adminKit.panelRadius,
  'bg-card p-3 text-card-foreground [--cell-size:2rem]'
);

export const calendarDayButtonClass = cn(
  adminKit.nestedRadius,
  'flex size-full items-center justify-center border-0 bg-transparent p-0 text-[13px] font-normal leading-none shadow-none outline-none',
  'ring-0 ring-offset-0 focus-visible:ring-0 focus-visible:ring-offset-0',
  'hover:bg-accent hover:text-accent-foreground',
  'data-[today=true]:font-semibold data-[today=true]:not([data-selected-single=true]):bg-accent/50',
  'data-[selected-single=true]:bg-primary data-[selected-single=true]:font-normal data-[selected-single=true]:text-primary-foreground',
  'data-[selected-single=true]:hover:bg-primary data-[selected-single=true]:hover:text-primary-foreground',
  'data-[range-start=true]:rounded-l-[8px] data-[range-start=true]:bg-primary data-[range-start=true]:text-primary-foreground',
  'data-[range-end=true]:rounded-r-[8px] data-[range-end=true]:bg-primary data-[range-end=true]:text-primary-foreground',
  'data-[range-middle=true]:rounded-none data-[range-middle=true]:bg-admin-selection data-[range-middle=true]:text-foreground'
);
