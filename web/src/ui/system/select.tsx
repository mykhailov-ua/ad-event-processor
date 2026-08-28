import { useEffect, useId, useRef, useState } from 'react';
import { cn } from '../../lib/cn.js';
import styles from './select.module.css';

export type SelectOption = {
  value: string;
  label: string;
};

export type SelectProps = {
  value: string;
  onChange: (value: string) => void;
  options: SelectOption[];
  id?: string;
  disabled?: boolean;
  placeholder?: string;
  'aria-label'?: string;
};

export function Select({
  value,
  onChange,
  options,
  id,
  disabled = false,
  placeholder = 'Select...',
  'aria-label': ariaLabel,
}: SelectProps) {
  const autoId = useId();
  const listboxId = id ?? autoId;
  const wrapRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);

  const selected = options.find((option) => option.value === value);
  const displayLabel = selected?.label ?? placeholder;

  useEffect(() => {
    if (!open) return;

    const onPointerDown = (event: MouseEvent) => {
      if (!wrapRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    };

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false);
      }
    };

    document.addEventListener('mousedown', onPointerDown);
    document.addEventListener('keydown', onKeyDown);
    return () => {
      document.removeEventListener('mousedown', onPointerDown);
      document.removeEventListener('keydown', onKeyDown);
    };
  }, [open]);

  const selectValue = (next: string) => {
    onChange(next);
    setOpen(false);
  };

  return (
    <div
      ref={wrapRef}
      className={styles.wrap}
      data-dropdown-open={open ? '' : undefined}
    >
      <button
        type="button"
        id={listboxId}
        className={cn(styles.trigger, open ? styles.triggerOpen : '')}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={ariaLabel}
        disabled={disabled}
        onClick={() => setOpen((prev) => !prev)}
      >
        <span className={styles.label}>{displayLabel}</span>
        <span className={styles.chevron} aria-hidden="true">
          v
        </span>
      </button>
      {open ? (
        <ul className={styles.drop} role="listbox" aria-labelledby={listboxId}>
          {options.map((option) => {
            const selectedOption = option.value === value;
            return (
              <li key={option.value} role="presentation">
                <button
                  type="button"
                  role="option"
                  aria-selected={selectedOption}
                  className={cn(styles.option, selectedOption ? styles.optionSelected : '')}
                  onClick={() => selectValue(option.value)}
                >
                  {option.label}
                </button>
              </li>
            );
          })}
        </ul>
      ) : null}
    </div>
  );
}
