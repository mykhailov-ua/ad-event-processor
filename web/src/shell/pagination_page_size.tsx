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
  layout?: 'stacked' | 'inline';
  onChange: (limit: number) => void;
};

export function PaginationPageSize({
  id,
  value,
  disabled = false,
  layout = 'stacked',
  onChange,
}: PaginationPageSizeProps) {
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

  if (layout === 'inline') {
    return (
      <div className="flex shrink-0 items-center gap-2">
        <label className="shrink-0 text-xs text-muted-foreground" htmlFor={id}>Per page</label>
        <Input
          id={id}
          type="number"
          inputMode="numeric"
          min={1}
          max={OPTIMAL_LIST_LIMIT_MAX}
          className="min-h-7 w-14 rounded-[5px] px-2 py-1 text-sm leading-[18px] tabular-nums [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
          disabled={disabled}
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          onBlur={() => commit(draft)}
          onKeyDown={handleKeyDown}
        />
      </div>
    );
  }

  return (
    <FilterField htmlFor={id} label="Per page" className="w-[5.5rem] shrink-0">
      <Input
        id={id}
        type="number"
        inputMode="numeric"
        min={1}
        max={OPTIMAL_LIST_LIMIT_MAX}
        className="min-h-7 py-1 text-sm leading-[18px] tabular-nums [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
        disabled={disabled}
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
        onBlur={() => commit(draft)}
        onKeyDown={handleKeyDown}
      />
    </FilterField>
  );
}
