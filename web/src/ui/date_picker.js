import { el } from '../lib/dom.js';
import { renderIcon } from './icon.js';
import { renderSelect } from './select.js';

const MONTH_NAMES = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
];
const DAY_NAMES = ['Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa', 'Su'];

/**
 * Format a Date object to YYYY-MM-DD HH:mm for display.
 *
 * @param {Date} date
 * @returns {string}
 */
export function formatDisplayDateTime(date) {
  if (!date || Number.isNaN(date.getTime())) return '';
  const pad = (n) => String(n).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

/**
 * Mount a custom, theme-aware Vanilla JS Date & Time Picker popover.
 *
 * @param {object} opts
 * @param {string} opts.id - Input element ID
 * @param {string} opts.value - Initial ISO string or YYYY-MM-DD HH:mm
 * @param {(isoValue: string) => void} opts.onChange - Callback on date/time change
 * @param {string} [opts.placeholder]
 * @returns {HTMLElement}
 */
export function createDatePicker(opts) {
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
    onClick: (e) => {
      e.stopPropagation();
      togglePopover();
    },
  });

  const iconEl = renderIcon('calendar', { size: 16, className: 'date-picker-trigger-icon' });

  const triggerEl = el('div', {
    className: 'date-picker-trigger',
    onClick: (e) => {
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
    onClick: (e) => e.stopPropagation(),
  });

  const wrapper = el('div', { className: 'custom-date-picker-wrapper' },
    triggerEl,
    popoverEl,
  );

  function togglePopover() {
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

  function closePopover() {
    open = false;
    popoverEl.style.display = 'none';
    document.removeEventListener('click', onDocClick);
    document.removeEventListener('keydown', onDocKey);
  }

  function onDocClick() {
    closePopover();
  }

  function onDocKey(e) {
    if (e.key === 'Escape') closePopover();
  }

  function updateValue(newDate) {
    selectedDate = newDate;
    inputEl.value = formatDisplayDateTime(selectedDate);
    opts.onChange(selectedDate.toISOString());
  }

  function renderPopover() {
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

    const isSameDay = (d1, d2) =>
      d1.getFullYear() === d2.getFullYear() &&
      d1.getMonth() === d2.getMonth() &&
      d1.getDate() === d2.getDate();

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

    // Time Selectors via custom renderSelect
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
      el('button', {
        type: 'button',
        className: 'btn btn--secondary btn--xs',
        onClick: () => {
          updateValue(new Date());
          closePopover();
        },
      }, 'Now'),
      el('button', {
        type: 'button',
        className: 'btn btn--primary btn--xs',
        onClick: () => closePopover(),
      }, 'Apply'),
    );

    popoverEl.replaceChildren(header, weekdaysRow, daysGrid, timeRow, footer);
  }

  return wrapper;
}
