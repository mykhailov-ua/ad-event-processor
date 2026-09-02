import { type KeyboardEvent, useEffect, useState } from 'react';

import { Input } from '@/components/ui/input';
import {
  clampListLimit,
  DEFAULT_LIST_LIMIT,
  OPTIMAL_LIST_LIMIT_MAX,
} from '@/lib/list_query';

import { FilterField } from './filter_panel';

export type PaginationPageSizeProps = {
  id: string;
  value: number;
  disabled?: boolean;
  onChange: (limit: number) => void;
};

export function PaginationPageSize({ id, value, disabled = false, onChange }: PaginationPageSizeProps) {
  const applied = value > 0 ? value : DEFAULT_LIST_LIMIT;
  const [draft, setDraft] = useState(String(applied));

  useEffect(() => {
    setDraft(String(applied));
  }, [applied]);

  function commit(raw: string) {
    const next = clampListLimit(Number.parseInt(raw, 10));
    setDraft(String(next));
    if (next !== applied) {
      onChange(next);
    }
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'Enter') {
      event.preventDefault();
      commit(draft);
    }
  }

  return (
    <FilterField htmlFor={id} label="Per page" className="w-[5.5rem] shrink-0">
      <Input
        id={id}
        type="number"
        inputMode="numeric"
        min={1}
        max={OPTIMAL_LIST_LIMIT_MAX}
        className="text-sm tabular-nums [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
        disabled={disabled}
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
        onBlur={() => commit(draft)}
        onKeyDown={handleKeyDown}
      />
    </FilterField>
  );
}
