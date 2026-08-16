import { useEffect, useRef, useState } from 'react';
import { Button } from './button.js';
import { Icon } from './icon.js';

/**
 * Format a Date object to YYYY-MM-DD HH:mm for display.
 */
export function formatDisplayDateTime(date: Date | null | undefined): string {
  if (!date || Number.isNaN(date.getTime())) return '';
  const pad = (n: number): string => String(n).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function parseIsoValue(iso: string): Date {
  if (!iso) return new Date();
  const parsed = new Date(iso);
  return Number.isNaN(parsed.getTime()) ? new Date() : parsed;
}

function isSameDay(d1: Date, d2: Date): boolean {
  return d1.getFullYear() === d2.getFullYear()
    && d1.getMonth() === d2.getMonth()
    && d1.getDate() === d2.getDate();
}

const MONTH_NAMES = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
];
const DAY_NAMES = ['Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa', 'Su'];

const HOUR_OPTS = Array.from({ length: 24 }, (_, i) => ({
  value: String(i),
  label: String(i).padStart(2, '0'),
}));

const MINUTE_OPTS = Array.from({ length: 12 }, (_, i) => {
  const m = i * 5;
  return { value: String(m), label: String(m).padStart(2, '0') };
});

export type DatePickerProps = {
  id: string;
  value: string;
  onChange: (isoValue: string) => void;
  placeholder?: string;
};

/**
 * Theme-aware date and time picker popover.
 */
export function DatePicker({
  id,
  value,
  onChange,
  placeholder = 'YYYY-MM-DD HH:mm',
}: DatePickerProps) {
  const selectedDate = parseIsoValue(value);
  const [open, setOpen] = useState(false);
  const [viewYear, setViewYear] = useState(() => selectedDate.getFullYear());
  const [viewMonth, setViewMonth] = useState(() => selectedDate.getMonth());
  const wrapperRef = useRef<HTMLDivElement>(null);

  const openPopover = () => {
    setViewYear(selectedDate.getFullYear());
    setViewMonth(selectedDate.getMonth());
    setOpen(true);
  };

  const updateValue = (next: Date) => {
    onChange(next.toISOString());
  };

  useEffect(() => {
    if (!open) return undefined;
    const onDocClick = (e: MouseEvent) => {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    const onDocKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false);
    };
    document.addEventListener('click', onDocClick);
    document.addEventListener('keydown', onDocKey);
    return () => {
      document.removeEventListener('click', onDocClick);
      document.removeEventListener('keydown', onDocKey);
    };
  }, [open]);

  const firstDayIndex = (new Date(viewYear, viewMonth, 1).getDay() + 6) % 7;
  const daysInMonth = new Date(viewYear, viewMonth + 1, 0).getDate();
  const today = new Date();

  const prevMonth = () => {
    if (viewMonth > 0) {
      setViewMonth(viewMonth - 1);
      return;
    }
    setViewMonth(11);
    setViewYear(viewYear - 1);
  };

  const nextMonth = () => {
    if (viewMonth < 11) {
      setViewMonth(viewMonth + 1);
      return;
    }
    setViewMonth(0);
    setViewYear(viewYear + 1);
  };

  return (
    <div ref={wrapperRef} className="custom-date-picker-wrapper">
      <div
        className="date-picker-trigger"
        onClick={(e) => {
          e.stopPropagation();
          if (open) setOpen(false);
          else openPopover();
        }}
      >
        <input
          id={id}
          type="text"
          readOnly
          className="form-input font-mono date-picker-input"
          value={formatDisplayDateTime(selectedDate)}
          placeholder={placeholder}
        />
        <Icon name="calendar" size={16} className="date-picker-trigger-icon" />
      </div>

      {open ? (
        <div
          className="custom-date-popover elevation-raised"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="cdp-header">
            <button type="button" className="cdp-nav-btn" onClick={prevMonth}>‹</button>
            <span className="cdp-month-label">{`${MONTH_NAMES[viewMonth]} ${viewYear}`}</span>
            <button type="button" className="cdp-nav-btn" onClick={nextMonth}>›</button>
          </div>

          <div className="cdp-weekdays">
            {DAY_NAMES.map((d) => (
              <span key={d} className="cdp-weekday">{d}</span>
            ))}
          </div>

          <div className="cdp-days">
            {Array.from({ length: firstDayIndex }, (_, i) => (
              <span key={`empty-${i}`} className="cdp-day cdp-day--empty" />
            ))}
            {Array.from({ length: daysInMonth }, (_, i) => {
              const day = i + 1;
              const dayDate = new Date(viewYear, viewMonth, day);
              const isSelected = isSameDay(dayDate, selectedDate);
              const isToday = isSameDay(dayDate, today);
              return (
                <button
                  key={day}
                  type="button"
                  className={[
                    'cdp-day',
                    isSelected ? 'cdp-day--selected' : '',
                    isToday ? 'cdp-day--today' : '',
                  ].filter(Boolean).join(' ')}
                  onClick={() => {
                    const next = new Date(selectedDate);
                    next.setFullYear(viewYear, viewMonth, day);
                    updateValue(next);
                  }}
                >
                  {day}
                </button>
              );
            })}
          </div>

          <div className="cdp-time-row">
            <span className="cdp-time-label">Time:</span>
            <select
              className="cdp-time-select"
              value={String(selectedDate.getHours())}
              aria-label="Hour"
              onChange={(e) => {
                const next = new Date(selectedDate);
                next.setHours(Number(e.target.value));
                updateValue(next);
              }}
            >
              {HOUR_OPTS.map((opt) => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </select>
            <span className="cdp-time-sep">:</span>
            <select
              className="cdp-time-select"
              value={String(Math.floor(selectedDate.getMinutes() / 5) * 5)}
              aria-label="Minute"
              onChange={(e) => {
                const next = new Date(selectedDate);
                next.setMinutes(Number(e.target.value));
                updateValue(next);
              }}
            >
              {MINUTE_OPTS.map((opt) => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </select>
          </div>

          <div className="cdp-footer">
            <Button
              label="Now"
              variant="secondary"
              className="btn--xs"
              onClick={() => {
                updateValue(new Date());
                setOpen(false);
              }}
            />
            <Button
              label="Apply"
              variant="primary"
              className="btn--xs"
              onClick={() => setOpen(false)}
            />
          </div>
        </div>
      ) : null}
    </div>
  );
}