import { useCallback, useState } from 'react';

// Controlled/uncontrolled primitive for overlays and pickers.
// value !== undefined => parent owns state; defaultValue seeds internal state when uncontrolled.
export function useControllableState<T>({
  value,
  defaultValue,
  onChange,
}: {
  value?: T;
  defaultValue?: T;
  onChange?: (next: T) => void;
}) {
  const [uncontrolled, setUncontrolled] = useState(defaultValue);
  const isControlled = value !== undefined;
  const state = isControlled ? value : uncontrolled;

  const setState = useCallback(
    (next: T) => {
      if (!isControlled) {
        setUncontrolled(next);
      }
      onChange?.(next);
    },
    [isControlled, onChange]
  );

  return [state, setState] as const;
}
