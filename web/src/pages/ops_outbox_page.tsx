import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';

import { listOpsOutbox } from '@/api/ops_api';
import { OpsOutbox } from '@/domains/ops/ops_outbox';
import { useResource } from '@/hooks/use_resource';
import { parseListLimit } from '@/lib/list_query';

const CURSOR_STACK_KEY = 'cursor_stack';

function parseCursorStack(raw: string | null): string[] {
  if (!raw) {
    return [];
  }
  try {
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed.filter((value): value is string => typeof value === 'string');
  } catch {
    return [];
  }
}

export function OpsOutboxPage() {
  const [searchParams, setSearchParams] = useSearchParams();

  const limit = parseListLimit(searchParams.get('limit'), 100);
  const cursor = searchParams.get('cursor') ?? undefined;
  const cursorStack = useMemo(
    () => parseCursorStack(searchParams.get(CURSOR_STACK_KEY)),
    [searchParams],
  );

  const { data, error, fetching } = useResource(
    (signal) => listOpsOutbox({ limit, cursor }, signal),
    [limit, cursor],
  );

  const updateCursors = useCallback(
    (nextCursor: string | undefined, nextStack: string[]) => {
      const next = new URLSearchParams(searchParams);
      next.set('limit', String(limit));
      if (nextCursor) {
        next.set('cursor', nextCursor);
      } else {
        next.delete('cursor');
      }
      if (nextStack.length > 0) {
        next.set(CURSOR_STACK_KEY, JSON.stringify(nextStack));
      } else {
        next.delete(CURSOR_STACK_KEY);
      }
      setSearchParams(next, { replace: true });
    },
    [limit, searchParams, setSearchParams],
  );

  const onNext = useCallback(() => {
    if (!data?.next_cursor) {
      return;
    }
    const nextStack = cursor ? [...cursorStack, cursor] : cursorStack;
    updateCursors(data.next_cursor, nextStack);
  }, [cursor, cursorStack, data?.next_cursor, updateCursors]);

  const onPrev = useCallback(() => {
    if (cursorStack.length === 0) {
      updateCursors(undefined, []);
      return;
    }
    const nextStack = cursorStack.slice(0, -1);
    const previousCursor = cursorStack[cursorStack.length - 1];
    updateCursors(previousCursor, nextStack);
  }, [cursorStack, updateCursors]);

  return (
    <OpsOutbox
      items={data?.items ?? []}
      nextCursor={data?.next_cursor}
      total={data?.total}
      limit={limit}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
      canGoPrev={cursorStack.length > 0 || Boolean(cursor)}
      onPrev={onPrev}
      onNext={onNext}
    />
  );
}
