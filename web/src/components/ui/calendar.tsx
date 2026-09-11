import * as React from 'react';
import { ChevronDown, ChevronLeft, ChevronRight } from 'lucide-react';
import { DayButton, DayPicker, getDefaultClassNames } from 'react-day-picker';

import {
  buildCalendarClassNames,
  calendarDayButtonClass,
  calendarRootClass,
} from '@/components/ui/calendar_class_names';
import { cn } from '@/lib/utils';

export type CalendarProps = React.ComponentProps<typeof DayPicker> & {
  variant?: 'default' | 'admin' | 'toolbar';
};

function Calendar({
  className,
  classNames,
  showOutsideDays = true,
  captionLayout = 'label',
  variant = 'default',
  formatters,
  components,
  ...props
}: CalendarProps) {
  const cellSize = variant === 'toolbar' ? '[--cell-size:1.75rem]' : '[--cell-size:2rem]';

  return (
    <DayPicker
      showOutsideDays={showOutsideDays}
      captionLayout={captionLayout}
      className={cn(calendarRootClass, cellSize, className)}
      classNames={{
        ...buildCalendarClassNames(captionLayout),
        ...classNames,
      }}
      formatters={{
        formatMonthDropdown: (date) => date.toLocaleString('default', { month: 'short' }),
        ...formatters,
      }}
      components={{
        Root: ({ className: rootClassName, rootRef, ...rootProps }) => (
          <div data-slot="calendar" ref={rootRef} className={cn(rootClassName)} {...rootProps} />
        ),
        Chevron: ({ className: chevronClassName, orientation, ...chevronProps }) => {
          const iconClass = cn('size-4 shrink-0', chevronClassName);
          if (orientation === 'left') {
            return <ChevronLeft className={iconClass} {...chevronProps} />;
          }
          if (orientation === 'right') {
            return <ChevronRight className={iconClass} {...chevronProps} />;
          }
          return <ChevronDown className={iconClass} {...chevronProps} />;
        },
        DayButton: (dayButtonProps) => (
          <CalendarDayButton {...dayButtonProps} />
        ),
        WeekNumber: ({ children, ...weekProps }) => (
          <td {...weekProps}>
            <div className="flex size-[var(--cell-size)] items-center justify-center text-center">
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
  ...props
}: React.ComponentProps<typeof DayButton>) {
  const ref = React.useRef<HTMLButtonElement>(null);

  React.useEffect(() => {
    if (modifiers.focused) {
      ref.current?.focus();
    }
  }, [modifiers.focused]);

  return (
    <button
      ref={ref}
      type="button"
      className={cn(className, calendarDayButtonClass)}
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
      data-today={modifiers.today}
      {...props}
    />
  );
}

export { Calendar, CalendarDayButton };
