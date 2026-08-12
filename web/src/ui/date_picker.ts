import { el } from '../lib/dom.js';
import { renderIcon } from './icon.js';
import { renderSelect } from './select.js';
import { renderButton } from './button.js';

const MONTH_NAMES = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
];
const DAY_NAMES = ['Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa', 'Su'];

/**
 * Format a Date object to YYYY-MM-DD HH:mm for display.
 */
export function formatDisplayDateTime(date: Date | null | undefined): string {
  if (!date || Number.isNaN(date.getTime())) return '';
  const pad = (n: number): string => String(n).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

export type DatePickerOpts = {
  id: string;
  value: string;
  onChange: (isoValue: string) => void;
  placeholder?: string;
};

/**
 * Mount a custom, theme-aware Vanilla JS Date & Time Picker popover.
 */
export function createDatePicker(opts: DatePickerOpts): HTMLElement {
  const initialDate = opts.value ? new Date(opts.value) : new Date();
  let selectedDate = Number.isNaN(initialDate.getTime()) ? new Date() : initialDate;
  let viewYear = selectedDate.getFullYear();
  let viewMonth = selectedDate.getMonth();
  let open = false;

  const inputEl = el('input', {
    id: opts.id,
    type: 'text',
    readOnly: true,
    className: 'form-input font-mono date-picker-input',
    value: formatDisplayDateTime(selectedDate),
    placeholder: opts.placeholder ?? 'YYYY-MM-DD HH:mm',
    onClick: (e: Event) => {
      e.stopPropagation();
      togglePopover();
    },
  }) as HTMLInputElement;

  const iconEl = renderIcon('calendar', { size: 16, className: 'date-picker-trigger-icon' });

  const triggerEl = el('div', {
    className: 'date-picker-trigger',
    onClick: (e: Event) => {
      e.stopPropagation();
      togglePopover();
    },
  },
    inputEl,
    iconEl,
  );

  const popoverEl = el('div', {
    className: 'custom-date-popover elevation-raised',
    style: { display: 'none' },
    onClick: (e: Event) => e.stopPropagation(),
  });

  const wrapper = el('div', { className: 'custom-date-picker-wrapper' },
    triggerEl,
    popoverEl,
  );

  function togglePopover(): void {
    open = !open;
    if (open) {
      viewYear = selectedDate.getFullYear();
      viewMonth = selectedDate.getMonth();
      renderPopover();
      popoverEl.style.display = 'block';
      document.addEventListener('click', onDocClick);
      document.addEventListener('keydown', onDocKey);
    } else {
      closePopover();
    }
  }

  function closePopover(): void {
    open = false;
    popoverEl.style.display = 'none';
    document.removeEventListener('click', onDocClick);
    document.removeEventListener('keydown', onDocKey);
  }

  function onDocClick(): void {
    closePopover();
  }

  function onDocKey(e: KeyboardEvent): void {
    if (e.key === 'Escape') closePopover();
  }

  function updateValue(newDate: Date): void {
    selectedDate = newDate;
    inputEl.value = formatDisplayDateTime(selectedDate);
    opts.onChange(selectedDate.toISOString());
  }

  function renderPopover(): void {
    const header = el('div', { className: 'cdp-header' },
      el('button', {
        type: 'button',
        className: 'cdp-nav-btn',
        onClick: () => {
          viewMonth -= 1;
          if (viewMonth < 0) {
            viewMonth = 11;
            viewYear -= 1;
          }
          renderPopover();
        },
      }, '‹'),
      el('span', { className: 'cdp-month-label' }, `${MONTH_NAMES[viewMonth]} ${viewYear}`),
      el('button', {
        type: 'button',
        className: 'cdp-nav-btn',
        onClick: () => {
          viewMonth += 1;
          if (viewMonth > 11) {
            viewMonth = 0;
            viewYear += 1;
          }
          renderPopover();
        },
      }, '›'),
    );

    const weekdaysRow = el('div', { className: 'cdp-weekdays' },
      DAY_NAMES.map((d) => el('span', { className: 'cdp-weekday' }, d)),
    );

    const firstDayIndex = (new Date(viewYear, viewMonth, 1).getDay() + 6) % 7;
    const daysInMonth = new Date(viewYear, viewMonth + 1, 0).getDate();
    const daysGrid = el('div', { className: 'cdp-days' });

    for (let i = 0; i < firstDayIndex; i++) {
      daysGrid.appendChild(el('span', { className: 'cdp-day cdp-day--empty' }));
    }

    const isSameDay = (d1: Date, d2: Date): boolean =>
      d1.getFullYear() === d2.getFullYear()
      && d1.getMonth() === d2.getMonth()
      && d1.getDate() === d2.getDate();

    const today = new Date();

    for (let day = 1; day <= daysInMonth; day++) {
      const dayDate = new Date(viewYear, viewMonth, day);
      const isSelected = isSameDay(dayDate, selectedDate);
      const isToday = isSameDay(dayDate, today);

      const dayBtn = el('button', {
        type: 'button',
        className: [
          'cdp-day',
          isSelected ? 'cdp-day--selected' : '',
          isToday ? 'cdp-day--today' : '',
        ].filter(Boolean).join(' '),
        onClick: () => {
          const next = new Date(selectedDate);
          next.setFullYear(viewYear, viewMonth, day);
          updateValue(next);
          renderPopover();
        },
      }, String(day));

      daysGrid.appendChild(dayBtn);
    }

    const hourOpts = Array.from({ length: 24 }, (_, i) => {
      const val = String(i);
      const pad = String(i).padStart(2, '0');
      return { value: val, label: pad };
    });

    const hoursSelect = renderSelect({
      value: String(selectedDate.getHours()),
      options: hourOpts,
      onChange: (val) => {
        const next = new Date(selectedDate);
        next.setHours(Number(val));
        updateValue(next);
      },
    });

    const minOpts = Array.from({ length: 12 }, (_, i) => {
      const val = String(i * 5);
      const pad = String(i * 5).padStart(2, '0');
      return { value: val, label: pad };
    });

    const minsSelect = renderSelect({
      value: String(Math.floor(selectedDate.getMinutes() / 5) * 5),
      options: minOpts,
      onChange: (val) => {
        const next = new Date(selectedDate);
        next.setMinutes(Number(val));
        updateValue(next);
      },
    });

    const timeRow = el('div', { className: 'cdp-time-row' },
      el('span', { className: 'cdp-time-label' }, 'Time:'),
      hoursSelect,
      el('span', { className: 'cdp-time-sep' }, ':'),
      minsSelect,
    );

    const footer = el('div', { className: 'cdp-footer' },
      renderButton({
        label: 'Now',
        variant: 'secondary',
        className: 'btn--xs',
        onClick: () => {
          updateValue(new Date());
          closePopover();
        },
      }),
      renderButton({
        label: 'Apply',
        variant: 'primary',
        className: 'btn--xs',
        onClick: () => closePopover(),
      }),
    );

    popoverEl.replaceChildren(header, weekdaysRow, daysGrid, timeRow, footer);
  }

  return wrapper;
}
