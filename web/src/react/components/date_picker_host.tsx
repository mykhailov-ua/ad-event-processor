import { useEffect, useRef } from 'react';
import { createDatePicker } from '../../ui/date_picker.js';

export type DatePickerHostProps = {
  id: string;
  value: string;
  onChange: (isoValue: string) => void;
  placeholder?: string;
};

/**
 * Mount the legacy date picker widget inside a React host.
 */
export function DatePickerHost({ id, value, onChange, placeholder }: DatePickerHostProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!ref.current) return undefined;
    const host = ref.current;
    host.replaceChildren(createDatePicker({ id, value, onChange, placeholder }));
    return () => host.replaceChildren();
  }, [id, value, onChange, placeholder]);

  return <div ref={ref} />;
}
