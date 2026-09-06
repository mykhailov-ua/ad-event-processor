import { type KeyboardEvent, useEffect, useState } from 'react';

import { Input } from '@/components/ui/input';
import { clampListLimit, DEFAULT_LIST_LIMIT, OPTIMAL_LIST_LIMIT_MAX } from '@/lib/list_query';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type PaginationPageSizeProps = {
  id: string;
  value: number;
  disabled?: boolean;
  onChange: (limit: number) => void;
};

export function PaginationPageSize({
  id,
  value,
  disabled = false,
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

  return (
    <div className="flex shrink-0 items-center gap-2">
      <label className={cn('shrink-0', adminKit.controlText, 'text-muted-foreground')} htmlFor={id}>
        Per page
      </label>
      <Input
        id={id}
        type="number"
        inputMode="numeric"
        min={1}
        max={OPTIMAL_LIST_LIMIT_MAX}
        className={cn(adminKit.controlHeight, 'w-14 px-2 py-0 tabular-nums', adminKit.controlText)}
        disabled={disabled}
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
        onBlur={() => commit(draft)}
        onKeyDown={handleKeyDown}
      />
    </div>
  );
}
